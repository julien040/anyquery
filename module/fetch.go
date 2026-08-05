package module

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/edsrzf/mmap-go"
	"github.com/hashicorp/go-hclog"
	"github.com/klauspost/compress/zstd"
)

// defaultMaxDownloadSize is the default cap on a remote fetch or the stdin
// spool, and independently on the decompressed output of any compressed
// source. It sits far above any honest data file, because a multi-GiB table is
// ordinary use: what it bounds is a stream that never ends, not a file someone
// deliberately asked for. Decompressed output is held to the same number rather
// than a tighter one, so that compressing a file never makes it unreadable at a
// size its plain form queries fine at.
const defaultMaxDownloadSize int64 = 32 << 30 // 32 GiB

// maxDownloadSizeEnv names the environment variable that overrides
// defaultMaxDownloadSize, so a cap that is wrong for a machine can be moved
// without a rebuild. Its value is a byte count with an optional unit suffix
// ("512MB", "64GiB"): a decimal suffix is a power of 1000, an "i" suffix and a
// bare letter are powers of 1024. An unusable value falls back to the default
// rather than to an unbounded fetch.
const maxDownloadSizeEnv = "ANYQUERY_MAX_DOWNLOAD_SIZE"

const maxRedirects = 10

// maxZstdWindow caps the zstd decoder's window. The window size is declared in
// the frame header, i.e. chosen by whoever produced the stream, and is
// allocated before any output byte is produced — so the output-size cap below
// would never fire on a frame that only asks for a huge window. 128 MiB is far
// above what any real compressor emits (level 19 uses 8 MiB).
const maxZstdWindow = 1 << 27

// Fetcher is the only place a Source (see source.go) is ever turned into
// bytes. Dispatch is on Source.Kind; there is no fallback branch — an
// unhandled kind is an internal error, not a guess.
type Fetcher struct {
	HTTP         *http.Client
	MaxBytes     int64  // default defaultMaxDownloadSize, or maxDownloadSizeEnv
	CacheDir     string // default xdg.CacheHome/anyquery/downloads
	Restrictions *Restrictions
}

// NewFetcher returns a Fetcher with the default transport, size cap, and
// cache directory, enforcing r (nil means unrestricted).
func NewFetcher(r *Restrictions) *Fetcher {
	return &Fetcher{HTTP: newFetchHTTPClient(), Restrictions: r}
}

func newFetchHTTPClient() *http.Client {
	transport := &http.Transport{
		ResponseHeaderTimeout: 15 * time.Second,
		// Our own decompression (for a ".gz"/".zst" file's *content*, not
		// HTTP transport-level compression) must see the exact bytes the
		// server sent. Go's Transport otherwise transparently gzip-decodes and
		// strips Content-Encoding when it added its own Accept-Encoding
		// header, which would silently double-decompress a .gz response
		// whose server happens to also set Content-Encoding: gzip.
		DisableCompression: true,
	}
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to unsupported scheme %q", req.URL.Scheme)
			}
			return nil
		},
	}
}

// maxBytes is the single cap every size-bounded path uses: the transfer and
// the decompressed output alike.
func (f *Fetcher) maxBytes() int64 {
	if f.MaxBytes > 0 {
		return f.MaxBytes
	}
	if n := envMaxDownloadSize(); n > 0 {
		return n
	}
	return defaultMaxDownloadSize
}

// envMaxDownloadSize resolves maxDownloadSizeEnv once per process: the cap must
// not change under a running query, and one warning about a malformed value is
// enough.
var envMaxDownloadSize = sync.OnceValue(func() int64 {
	return resolveMaxDownloadSize(os.Getenv(maxDownloadSizeEnv))
})

// resolveMaxDownloadSize turns a raw maxDownloadSizeEnv value into a cap, and
// 0 when there is none to apply. An unusable value is reported and treated as
// unset, so it falls back to the defaults and never to an unbounded fetch.
func resolveMaxDownloadSize(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := parseByteSize(raw)
	if err != nil {
		hclog.Default().Warn("fetch: ignoring "+maxDownloadSizeEnv, "value", raw, "error", err)
		return 0
	}
	return n
}

// byteSizeUnits are matched against the end of a size string, longest form
// first, so "KiB" is never read as the shorter "B" and "KB" never as "B".
var byteSizeUnits = []struct {
	suffix string
	scale  int64
}{
	{"KIB", 1 << 10}, {"MIB", 1 << 20}, {"GIB", 1 << 30}, {"TIB", 1 << 40},
	{"KB", 1_000}, {"MB", 1_000_000}, {"GB", 1_000_000_000}, {"TB", 1_000_000_000_000},
	{"K", 1 << 10}, {"M", 1 << 20}, {"G", 1 << 30}, {"T", 1 << 40},
	{"B", 1},
}

// parseByteSize reads a whole number of bytes, with an optional unit suffix
// (see maxDownloadSizeEnv). A fractional count is rejected rather than rounded:
// "1.5GB" is spelled 1500MB.
func parseByteSize(v string) (int64, error) {
	s := strings.ToUpper(strings.TrimSpace(v))
	scale := int64(1)
	for _, u := range byteSizeUnits {
		if strings.HasSuffix(s, u.suffix) {
			scale = u.scale
			s = strings.TrimSpace(strings.TrimSuffix(s, u.suffix))
			break
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a whole number of bytes", v)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%q must be greater than zero", v)
	}
	if n > math.MaxInt64/scale {
		return 0, fmt.Errorf("%q is larger than the maximum representable size", v)
	}
	return n * scale, nil
}

func (f *Fetcher) cacheDir() string {
	if f.CacheDir != "" {
		return f.CacheDir
	}
	return filepath.Join(xdg.CacheHome, "anyquery", "downloads")
}

func (f *Fetcher) httpClient() *http.Client {
	if f.HTTP != nil {
		return f.HTTP
	}
	return newFetchHTTPClient()
}

// codec is the content compression a source must be decoded through before any
// reader sees it. It describes the *content* of the file (a .csv.gz holding
// CSV), not an HTTP transfer encoding — although a server may announce the
// same thing in Content-Encoding, which is why both are mapped to this type.
type codec uint8

const (
	codecNone codec = iota
	codecGzip
	codecZstd
)

// codecForPath reports the codec implied by a file path's (or URL path's)
// extension. Extension matching is the only detection for local files: there
// is no content sniffing, so a reader never has to guess.
func codecForPath(p string) codec {
	lower := strings.ToLower(p)
	switch {
	case strings.HasSuffix(lower, ".gz"):
		return codecGzip
	case strings.HasSuffix(lower, ".zst"), strings.HasSuffix(lower, ".zstd"):
		return codecZstd
	}
	return codecNone
}

// codecForContentEncoding maps a Content-Encoding header value to a codec.
// Anything else (including "identity" and an empty header) is codecNone, so an
// unknown encoding is handed to the reader untouched rather than mis-decoded.
func codecForContentEncoding(v string) codec {
	switch {
	case strings.EqualFold(v, "gzip"):
		return codecGzip
	case strings.EqualFold(v, "zstd"):
		return codecZstd
	}
	return codecNone
}

// newDecompressor wraps r in the decoder for c. codecNone is a programming
// error: callers must check for it and use r directly.
func newDecompressor(c codec, r io.Reader) (io.ReadCloser, error) {
	switch c {
	case codecGzip:
		gz, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("fetch: not a valid gzip stream: %w", err)
		}
		return gz, nil
	case codecZstd:
		// Concurrency 1 keeps a reader that is abandoned mid-stream (because
		// the size cap fired) from leaving decode goroutines behind.
		zr, err := zstd.NewReader(r,
			zstd.WithDecoderMaxWindow(maxZstdWindow),
			zstd.WithDecoderConcurrency(1))
		if err != nil {
			return nil, fmt.Errorf("fetch: not a valid zstd stream: %w", err)
		}
		return zr.IOReadCloser(), nil
	default:
		return nil, fmt.Errorf("fetch: internal error: no decompressor for codec %d", c)
	}
}

// sizeCapError reports a cap being hit, naming the variable that moves it: the
// byte count alone leaves a user with a legitimately huge file no way forward.
func sizeCapError(label string, limit int64) error {
	return fmt.Errorf("fetch: %s exceeds the maximum download size of %d bytes (raise it with %s)",
		label, limit, maxDownloadSizeEnv)
}

// cappedReadCloser streams at most limit bytes and then errors. It is the
// streaming counterpart of writeTempCounted's cap: a compression bomb reaches
// callers of Open as a plain io.Reader, so the guard has to live in the reader
// itself. Exceeding the cap is always an error and never a short read that
// would look like a legitimate end of file.
type cappedReadCloser struct {
	r     io.Reader
	n     int64
	limit int64
	label string
	// closers are closed in order: the decoder before the file it reads from.
	closers []io.Closer
}

func (c *cappedReadCloser) Read(p []byte) (int, error) {
	if remaining := c.limit - c.n; remaining < int64(len(p)) {
		if remaining < 0 {
			remaining = 0
		}
		// One byte past the budget is enough to detect the overflow, and it is
		// trimmed off below so the caller never receives it.
		p = p[:remaining+1]
	}
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.n > c.limit {
		if n > 0 {
			n--
		}
		return n, sizeCapError(c.label, c.limit)
	}
	return n, err
}

func (c *cappedReadCloser) Close() error {
	var first error
	for _, cl := range c.closers {
		if err := cl.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Open returns a reader for s, respecting ttl for the remote cache. Callers
// must Close the result. A compressed source (see codecForPath) is decoded
// transparently, so every reader module receives plain content.
func (f *Fetcher) Open(s Source, ttl time.Duration) (io.ReadCloser, error) {
	switch s.Kind {
	case KindLocal:
		file, err := f.Restrictions.OpenLocal(s.Path)
		if err != nil {
			return nil, redactSourceError(s, err)
		}
		c := codecForPath(s.Path)
		if c == codecNone {
			return file, nil
		}
		dec, err := newDecompressor(c, file)
		if err != nil {
			file.Close()
			return nil, redactSourceError(s, err)
		}
		return &cappedReadCloser{
			r:       dec,
			limit:   f.maxBytes(),
			label:   "decompressed file",
			closers: []io.Closer{dec, file},
		}, nil
	case KindStdin:
		if err := f.Restrictions.Check(s); err != nil {
			return nil, err
		}
		return io.NopCloser(os.Stdin), nil
	case KindHTTP:
		if err := f.Restrictions.Check(s); err != nil {
			return nil, err
		}
		path, err := f.fetchToCache(s, ttl)
		if err != nil {
			return nil, redactSourceError(s, err)
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, redactSourceError(s, err)
		}
		return file, nil
	default:
		return nil, fmt.Errorf("fetch: internal error: unhandled source kind %d", s.Kind)
	}
}

// OpenMmap memory-maps s, respecting ttl for the remote cache. Since a mapping
// needs a real file, anything that is not already one on disk in plain form —
// a remote source, stdin, a compressed local file — is materialized in the
// cache directory first.
func (f *Fetcher) OpenMmap(s Source, ttl time.Duration) (mmap.MMap, error) {
	switch s.Kind {
	case KindLocal:
		if c := codecForPath(s.Path); c != codecNone {
			path, err := f.decompressLocalToCache(s, c)
			if err != nil {
				return nil, redactSourceError(s, err)
			}
			m, err := mmapPath(path)
			if err != nil {
				return nil, redactSourceError(s, err)
			}
			return m, nil
		}
		file, err := f.Restrictions.OpenLocal(s.Path)
		if err != nil {
			return nil, redactSourceError(s, err)
		}
		defer file.Close()
		m, err := mmap.Map(file, mmap.RDONLY, 0)
		if err != nil {
			return nil, redactSourceError(s, err)
		}
		return m, nil
	case KindStdin:
		if err := f.Restrictions.Check(s); err != nil {
			return nil, err
		}
		return f.mmapStdin()
	case KindHTTP:
		if err := f.Restrictions.Check(s); err != nil {
			return nil, err
		}
		path, err := f.fetchToCache(s, ttl)
		if err != nil {
			return nil, redactSourceError(s, err)
		}
		m, err := mmapPath(path)
		if err != nil {
			return nil, redactSourceError(s, err)
		}
		return m, nil
	default:
		return nil, fmt.Errorf("fetch: internal error: unhandled source kind %d", s.Kind)
	}
}

// mmapPath maps a file that is already plain content on disk. The descriptor is
// closed straight away: the mapping keeps its own reference to the underlying
// object on every platform mmap-go supports.
func mmapPath(path string) (mmap.MMap, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return mmap.Map(file, mmap.RDONLY, 0)
}

// mmapStdin spools stdin into the cache directory and maps that, since stdin is
// not seekable and cannot be mapped directly. The spool file is unlinked as
// soon as it is mapped: on POSIX the mapping stays valid afterwards, and on a
// platform where the unlink fails while mapped the file is left in the cache
// directory that clear_file_cache() empties.
func (f *Fetcher) mmapStdin() (mmap.MMap, error) {
	dir := f.cacheDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("fetch: creating cache directory: %w", err)
	}
	path, err := writeTempCounted(dir, os.Stdin, f.maxBytes(), "stdin")
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("fetch: spooling stdin: %w", err)
	}
	if info.Size() == 0 {
		os.Remove(path)
		return nil, fmt.Errorf("fetch: stdin is empty; nothing to memory-map")
	}
	m, err := mmapPath(path)
	os.Remove(path)
	if err != nil {
		return nil, fmt.Errorf("fetch: memory-mapping stdin: %w", err)
	}
	return m, nil
}

// decompressLocalToCache materializes the decompressed form of a compressed
// local file in the cache directory and returns its path, so it can be mapped.
// The source is opened through OpenLocal, keeping the sandbox containment that
// every local read goes through.
//
// The cache key covers the file's identity *and* its version (mtime and size),
// so editing the source file misses the cache instead of serving a stale
// decompression; the entries are reclaimed by clear_file_cache() like any other
// download.
func (f *Fetcher) decompressLocalToCache(s Source, c codec) (string, error) {
	file, err := f.Restrictions.OpenLocal(s.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// Stat the open descriptor, not the path: the key must describe the exact
	// bytes that are about to be decompressed.
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("fetch: stat local file: %w", err)
	}

	dir := f.cacheDir()
	cachePath := filepath.Join(dir, localCacheKey(s.Path, info))
	if cached, err := os.Stat(cachePath); err == nil && cached.Size() > 0 {
		return cachePath, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("fetch: creating cache directory: %w", err)
	}

	dec, err := newDecompressor(c, file)
	if err != nil {
		return "", err
	}
	defer dec.Close()
	tmpPath, err := writeTempCounted(dir, dec, f.maxBytes(), "decompressed file")
	if err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, cachePath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("fetch: finalizing decompression: %w", err)
	}
	return cachePath, nil
}

// localCacheKey names the cache entry holding the decompressed form of a local
// file. It is deliberately in the same namespace as cacheKey (a sha256 hex
// digest) but built from a distinct "local:" prefix, so a local entry can never
// collide with a downloaded URL's entry.
func localCacheKey(path string, info os.FileInfo) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	sum := sha256.Sum256(fmt.Appendf(nil, "local:%s:%d:%d", abs, info.ModTime().UnixNano(), info.Size()))
	return hex.EncodeToString(sum[:])
}

// cacheMeta holds the HTTP validators of a cached download, stored in a
// "<cache entry>.meta" sidecar. They are what makes a stale entry revalidatable
// with a conditional GET instead of a full re-download.
type cacheMeta struct {
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

func (m cacheMeta) empty() bool {
	return m.ETag == "" && m.LastModified == ""
}

func cacheMetaPath(cachePath string) string {
	return cachePath + ".meta"
}

// readCacheMeta returns the validators recorded for a cached body. A missing or
// unreadable sidecar yields the zero value, which means "re-download
// unconditionally" — never an error, since a lost sidecar only costs bandwidth.
func readCacheMeta(cachePath string) cacheMeta {
	data, err := os.ReadFile(cacheMetaPath(cachePath))
	if err != nil {
		return cacheMeta{}
	}
	var m cacheMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return cacheMeta{}
	}
	return m
}

// writeCacheMeta records the validators of a fresh 200 next to its body. It
// must be called *after* the body is renamed into place: a sidecar describing a
// body that is not the one on disk would make a later 304 reuse the wrong
// bytes, so on any failure the sidecar is removed rather than left behind.
// Failing to write it is not fatal — it only means the next refresh downloads
// the whole body again.
func writeCacheMeta(cachePath string, m cacheMeta) {
	metaPath := cacheMetaPath(cachePath)
	if m.empty() {
		os.Remove(metaPath)
		return
	}
	data, err := json.Marshal(m)
	if err != nil {
		os.Remove(metaPath)
		return
	}
	if err := os.WriteFile(metaPath, data, 0o600); err != nil {
		os.Remove(metaPath)
	}
}

// fetchToCache ensures a remote Source is present in the cache directory,
// fresh within ttl, and returns its path. Only KindHTTP reaches this — local
// sources bypass the cache entirely.
//
// A fresh entry never touches the network. A stale entry is revalidated when
// the previous response left validators behind: a 304 refreshes the entry's
// freshness stamp and reuses the bytes already on disk.
func (f *Fetcher) fetchToCache(s Source, ttl time.Duration) (string, error) {
	dir := f.cacheDir()
	cachePath := filepath.Join(dir, f.cacheKey(s))

	var validators cacheMeta
	if info, err := os.Stat(cachePath); err == nil && info.Size() > 0 {
		if time.Since(info.ModTime()) < ttl {
			return cachePath, nil
		}
		validators = readCacheMeta(cachePath)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("fetch: creating cache directory: %w", err)
	}

	got, err := f.fetchHTTP(s.URL, validators)
	if err != nil {
		return "", err
	}
	if got.notModified {
		now := time.Now()
		if err := os.Chtimes(cachePath, now, now); err == nil {
			return cachePath, nil
		}
		// The entry disappeared between the stat above and here, so there is
		// nothing left to revalidate against: download it outright.
		got, err = f.fetchHTTP(s.URL, cacheMeta{})
		if err != nil {
			return "", err
		}
	}
	defer got.body.Close()

	rawPath, err := writeTempCounted(dir, got.body, f.maxBytes(), "response")
	if err != nil {
		return "", err
	}

	finalPath := rawPath
	if got.codec != codecNone {
		decompressedPath, err := decompressToTemp(dir, rawPath, f.maxBytes(), got.codec)
		os.Remove(rawPath)
		if err != nil {
			return "", err
		}
		finalPath = decompressedPath
	}

	if err := os.Rename(finalPath, cachePath); err != nil {
		os.Remove(finalPath)
		return "", fmt.Errorf("fetch: finalizing download: %w", err)
	}
	writeCacheMeta(cachePath, got.meta)
	return cachePath, nil
}

// writeTempCounted copies up to limit+1 bytes from src into a fresh 0600 temp
// file in dir, fsyncs it, and returns its path; the caller renames it into
// place (an atomic publish), or removes it on error. Exceeding
// limit is an error, never a silent truncation. label distinguishes the raw
// download from the decompression step in the resulting error message (the
// decompressed-size cap is a distinct, independent check from the
// compressed-size one — see decompressToTemp).
func writeTempCounted(dir string, src io.Reader, limit int64, label string) (path string, err error) {
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return "", fmt.Errorf("fetch: creating temp file: %w", err)
	}
	path = tmp.Name()
	defer func() {
		if cerr := tmp.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("fetch: closing temp file: %w", cerr)
		}
		if err != nil {
			os.Remove(path)
		}
	}()
	if cerr := tmp.Chmod(0o600); cerr != nil {
		err = fmt.Errorf("fetch: chmod temp file: %w", cerr)
		return "", err
	}
	n, cerr := io.Copy(tmp, io.LimitReader(src, limit+1))
	if cerr != nil {
		err = fmt.Errorf("fetch: downloading: %w", cerr)
		return "", err
	}
	if n > limit {
		err = sizeCapError(label, limit)
		return "", err
	}
	if cerr := tmp.Sync(); cerr != nil {
		err = fmt.Errorf("fetch: fsync: %w", cerr)
		return "", err
	}
	return path, nil
}

// decompressToTemp decodes the file at rawPath with c into a fresh temp file in
// dir, capped at limit independently of the compressed size — a compression
// bomb guard, since a few hundred compressed bytes can expand without bound.
func decompressToTemp(dir, rawPath string, limit int64, c codec) (string, error) {
	raw, err := os.Open(rawPath)
	if err != nil {
		return "", fmt.Errorf("fetch: reopening downloaded file: %w", err)
	}
	defer raw.Close()
	dec, err := newDecompressor(c, raw)
	if err != nil {
		return "", err
	}
	defer dec.Close()
	return writeTempCounted(dir, dec, limit, "decompressed response")
}

// cacheKey is the sha256 of the rewritten URL with credential query
// parameters removed, so two presigned URLs for one object share a cache entry
// and no signature ever appears in a cache filename.
func (f *Fetcher) cacheKey(s Source) string {
	u := redactedURL(s.URL)
	sum := sha256.Sum256([]byte(u.String()))
	return hex.EncodeToString(sum[:])
}

var credentialQueryParams = []string{"aws_access_key_id", "aws_access_key_secret"}

// redactedURL returns orig without its userinfo, the credentialQueryParams, or
// any "X-Amz-*" parameter. Presigned URLs are plain https and carry their
// signature (X-Amz-Signature, X-Amz-Credential, …) in the query, which both
// expires and identifies the caller: it must not become part of a cache key,
// or every re-presign of one object would miss the cache and re-download it.
func redactedURL(orig *url.URL) *url.URL {
	u := *orig
	q := u.Query()
	for _, p := range credentialQueryParams {
		q.Del(p)
	}
	for k := range q {
		if strings.HasPrefix(strings.ToLower(k), "x-amz-") {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	u.User = nil
	return &u
}

// redactSourceError strips credential query-parameter values out of err's
// message before it ever reaches a SQL client: transport errors routinely echo
// the URL, and the query of a presigned URL carries the caller's signature and
// access-key id.
func redactSourceError(s Source, err error) error {
	if err == nil || s.URL == nil {
		return err
	}
	msg := err.Error()
	q := s.URL.Query()
	for _, p := range credentialQueryParams {
		if v := q.Get(p); v != "" {
			msg = strings.ReplaceAll(msg, v, "REDACTED")
		}
	}
	for k, vs := range q {
		if !strings.HasPrefix(strings.ToLower(k), "x-amz-") {
			continue
		}
		for _, v := range vs {
			if v != "" {
				msg = strings.ReplaceAll(msg, v, "REDACTED")
			}
		}
	}
	if msg == err.Error() {
		return err
	}
	return errors.New(msg)
}

// httpFetch is the outcome of one GET: either a 304, carrying no body, or a
// 200 whose body must be decoded through codec and whose validators go into the
// cache sidecar.
type httpFetch struct {
	body        io.ReadCloser
	codec       codec
	meta        cacheMeta
	notModified bool
}

// fetchHTTP performs the GET for a KindHTTP source — the only remote transport.
// A non-empty validators makes it a conditional request, so an unchanged
// resource comes back as a 304 with no body.
func (f *Fetcher) fetchHTTP(u *url.URL, validators cacheMeta) (httpFetch, error) {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return httpFetch{}, fmt.Errorf("fetch: building request: %w", err)
	}
	// Both validators are sent when both are known; per RFC 9110 a server that
	// understands entity tags gives If-None-Match precedence. Neither value is
	// ever synthesized — they are echoed back exactly as a previous 200 sent
	// them, or omitted.
	if validators.ETag != "" {
		req.Header.Set("If-None-Match", validators.ETag)
	}
	if validators.LastModified != "" {
		req.Header.Set("If-Modified-Since", validators.LastModified)
	}

	resp, err := f.httpClient().Do(req)
	if err != nil {
		return httpFetch{}, fmt.Errorf("fetch: %w", err)
	}
	// A 304 is only meaningful as the answer to a request we made conditional;
	// otherwise it falls through to the unexpected-status error below.
	if resp.StatusCode == http.StatusNotModified && !validators.empty() {
		resp.Body.Close()
		return httpFetch{notModified: true}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return httpFetch{}, fmt.Errorf("fetch: unexpected status %s", resp.Status)
	}

	// The *final* URL after redirects is what names the content: a shortener or
	// a signed-redirect endpoint routinely has no extension of its own.
	finalPath := u.Path
	if resp.Request != nil && resp.Request.URL != nil {
		finalPath = resp.Request.URL.Path
	}
	c := codecForContentEncoding(resp.Header.Get("Content-Encoding"))
	if c == codecNone {
		c = codecForPath(finalPath)
	}
	return httpFetch{
		body:  resp.Body,
		codec: c,
		meta: cacheMeta{
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
		},
	}, nil
}
