package module

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
)

// gzipBytes and zstdBytes build compressed fixtures in memory, so the
// compression tests never depend on a checked-in binary blob.
func gzipBytes(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(content)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func zstdBytes(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("zstd writer: %v", err)
	}
	if _, err := enc.Write([]byte(content)); err != nil {
		t.Fatalf("zstd write: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("zstd close: %v", err)
	}
	return buf.Bytes()
}

// readAllSource opens s and drains it, so a cap that only fires mid-stream is
// still observed.
func readAllSource(t *testing.T, f *Fetcher, s Source) (string, error) {
	t.Helper()
	rc, err := f.Open(s, time.Hour)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	return string(data), err
}

// TestCacheKeyExcludesCredentials: two presigned URLs for the same object
// differ in their signature and expiry, so keying the cache on the raw query
// would make every re-presign a cache miss (and would write the signature into
// a filename on disk).
func TestCacheKeyExcludesCredentials(t *testing.T) {
	f := NewFetcher(nil)
	s1 := mustParse(t, "https://bucket.s3.amazonaws.com/key.csv?X-Amz-Credential=AKIAAAA&X-Amz-Signature=AAA&X-Amz-Expires=900")
	s2 := mustParse(t, "https://bucket.s3.amazonaws.com/key.csv?X-Amz-Credential=AKIABBB&X-Amz-Signature=BBB&X-Amz-Expires=900")
	if f.cacheKey(s1) != f.cacheKey(s2) {
		t.Fatalf("cache key differs when only the presigned credentials differ")
	}

	// The legacy credential parameter names are stripped too.
	s3 := mustParse(t, "https://bucket.s3.amazonaws.com/key.csv?aws_access_key_id=AAA&aws_access_key_secret=BBB")
	s4 := mustParse(t, "https://bucket.s3.amazonaws.com/key.csv?aws_access_key_id=CCC&aws_access_key_secret=DDD")
	if f.cacheKey(s3) != f.cacheKey(s4) {
		t.Fatalf("cache key differs when only credentialQueryParams differ")
	}
}

func TestRedactSourceErrorStripsCredentials(t *testing.T) {
	s := mustParse(t, "https://bucket.s3.amazonaws.com/key.csv?X-Amz-Credential=AKIA&X-Amz-Signature=SUPERSECRET")
	err := fmt.Errorf("failed with secret SUPERSECRET embedded")
	redacted := redactSourceError(s, err)
	if strings.Contains(redacted.Error(), "SUPERSECRET") {
		t.Fatalf("presigned signature leaked in error: %v", redacted)
	}

	s = mustParse(t, "https://bucket.s3.amazonaws.com/key.csv?aws_access_key_id=AAA&aws_access_key_secret=OTHERSECRET")
	redacted = redactSourceError(s, fmt.Errorf("failed with secret OTHERSECRET embedded"))
	if strings.Contains(redacted.Error(), "OTHERSECRET") {
		t.Fatalf("credentialQueryParams value leaked in error: %v", redacted)
	}
}

func TestFetchHTTPRedirectLimit(t *testing.T) {
	const hops = 12
	mux := http.NewServeMux()
	for i := hops; i > 0; i-- {
		i := i
		mux.HandleFunc(fmt.Sprintf("/hop%d", i), func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, fmt.Sprintf("/hop%d", i-1), http.StatusFound)
		})
	}
	mux.HandleFunc("/hop0", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("done"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	s := mustParse(t, srv.URL+fmt.Sprintf("/hop%d", hops))
	if _, err := f.Open(s, time.Hour); err == nil {
		t.Fatalf("expected a redirect-limit error")
	}
}

func TestFetchHTTPRedirectNonHTTPScheme(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/badredirect", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "ftp://example.com/file")
		w.WriteHeader(http.StatusFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	s := mustParse(t, srv.URL+"/badredirect")
	_, err := f.Open(s, time.Hour)
	if err == nil {
		t.Fatalf("expected a redirect-scheme error")
	}
	if !strings.Contains(err.Error(), "unsupported scheme") {
		t.Fatalf("got %v, want an unsupported-scheme error", err)
	}
}

func TestFetchHTTPMaxBytes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte("a"), 1000))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	f.MaxBytes = 100
	s := mustParse(t, srv.URL+"/big")
	_, err := f.Open(s, time.Hour)
	if err == nil {
		t.Fatalf("expected a max-download-size error")
	}
	if !strings.Contains(err.Error(), "maximum download size") {
		t.Fatalf("got %v, want a max-size error", err)
	}
}

func TestFetchHTTPGzipDecompressedCap(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bomb.gz", func(w http.ResponseWriter, r *http.Request) {
		gz := gzip.NewWriter(w)
		gz.Write(bytes.Repeat([]byte("a"), 200_000)) // compresses to well under 1000 bytes
		gz.Close()
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	f.MaxBytes = 1000
	s := mustParse(t, srv.URL+"/bomb.gz")
	_, err := f.Open(s, time.Hour)
	if err == nil {
		t.Fatalf("expected the decompressed-size cap to trigger")
	}
	if !strings.Contains(err.Error(), "decompressed") {
		t.Fatalf("got %v, want a decompressed-size error", err)
	}
}

func TestFetchHTTPGzipContent(t *testing.T) {
	want := "a,b\n1,2\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/data.csv.gz", func(w http.ResponseWriter, r *http.Request) {
		gz := gzip.NewWriter(w)
		gz.Write([]byte(want))
		gz.Close()
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	s := mustParse(t, srv.URL+"/data.csv.gz")
	rc, err := f.Open(s, time.Hour)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFetchLocalBypassesCache(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "f.txt", "hello")
	cacheDir := t.TempDir()
	f := NewFetcher(&Restrictions{AllowedDirs: []string{dir}})
	f.CacheDir = cacheDir
	s := mustParse(t, p)

	rc, err := f.Open(s, time.Hour)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	rc.Close()

	entries, err := os.ReadDir(cacheDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("a local source wrote %d entries into the cache dir", len(entries))
	}
}

func TestFetchStdinDeniedUnderSandbox(t *testing.T) {
	s := mustParse(t, "stdin")

	f := NewFetcher(&Restrictions{})
	if _, err := f.Open(s, time.Hour); err == nil {
		t.Fatalf("stdin was permitted under a sandbox")
	}

	f2 := NewFetcher(nil)
	rc, err := f2.Open(s, time.Hour)
	if err != nil {
		t.Fatalf("stdin denied without a sandbox: %v", err)
	}
	rc.Close()
}

// TestFetchHTTPRequiresAllowRemote: a remote Source must never be fetched
// unless the policy explicitly allows it, regardless of Kind.
func TestFetchHTTPRequiresAllowRemote(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/f.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("secret"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	s := mustParse(t, srv.URL+"/f.txt")

	f := NewFetcher(&Restrictions{AllowRemote: false})
	f.CacheDir = t.TempDir()
	if _, err := f.Open(s, time.Hour); err == nil {
		t.Fatalf("remote fetch was permitted with AllowRemote: false")
	}

	f2 := NewFetcher(&Restrictions{AllowRemote: true})
	f2.CacheDir = t.TempDir()
	rc, err := f2.Open(s, time.Hour)
	if err != nil {
		t.Fatalf("remote fetch denied with AllowRemote: true: %v", err)
	}
	rc.Close()
}

// TestFetchPresignedErrorDoesNotLeakSignature drives the real Fetcher.Open
// return path rather than calling redactSourceError with a synthetic error:
// net/http embeds the full request URL (query included) in a dial error, so a
// presigned URL's signature reaches the caller unless Open's own
// redactSourceError wrapping strips it. The server is closed before the fetch
// so the dial fails offline and deterministically.
func TestFetchPresignedErrorDoesNotLeakSignature(t *testing.T) {
	srv := httptest.NewServer(http.NewServeMux())
	url := srv.URL + "/key.csv?X-Amz-Credential=AKIA&X-Amz-Signature=SUPERSECRET"
	srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	s := mustParse(t, url)

	_, err := f.Open(s, time.Hour)
	if err == nil {
		t.Fatalf("expected a dial error against a closed server")
	}
	if strings.Contains(err.Error(), "SUPERSECRET") {
		t.Fatalf("presigned signature leaked through the real Fetcher.Open path: %v", err)
	}
}

func TestFetchCacheFreshness(t *testing.T) {
	var counter int32
	mux := http.NewServeMux()
	mux.HandleFunc("/f.txt", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&counter, 1)
		fmt.Fprintf(w, "version-%d", n)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	s := mustParse(t, srv.URL+"/f.txt")

	rc1, err := f.Open(s, time.Hour)
	if err != nil {
		t.Fatalf("Open (1): %v", err)
	}
	data1, _ := io.ReadAll(rc1)
	rc1.Close()

	rc2, err := f.Open(s, time.Hour)
	if err != nil {
		t.Fatalf("Open (2): %v", err)
	}
	data2, _ := io.ReadAll(rc2)
	rc2.Close()
	if string(data1) != string(data2) {
		t.Fatalf("cache did not reuse a fresh entry: %q vs %q", data1, data2)
	}

	time.Sleep(10 * time.Millisecond)
	rc3, err := f.Open(s, time.Millisecond)
	if err != nil {
		t.Fatalf("Open (3): %v", err)
	}
	data3, _ := io.ReadAll(rc3)
	rc3.Close()
	if string(data3) == string(data1) {
		t.Fatalf("a stale cache entry was reused past its ttl")
	}
}

// TestOpenLocalGzipContent: a local .gz must be decoded on the way out, exactly
// like a remote one — readers only ever see plain content.
func TestOpenLocalGzipContent(t *testing.T) {
	want := "a,b\n1,2\n"
	dir := t.TempDir()
	p := writeTempFile(t, dir, "data.csv.gz", string(gzipBytes(t, want)))

	f := NewFetcher(&Restrictions{AllowedDirs: []string{dir}})
	f.CacheDir = t.TempDir()
	got, err := readAllSource(t, f, mustParse(t, p))
	if err != nil {
		t.Fatalf("reading a local .gz: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestOpenLocalZstdContent is the .zst counterpart of TestOpenLocalGzipContent.
func TestOpenLocalZstdContent(t *testing.T) {
	want := "a,b\n1,2\n"
	dir := t.TempDir()

	for _, name := range []string{"data.csv.zst", "data.csv.zstd"} {
		t.Run(name, func(t *testing.T) {
			p := writeTempFile(t, dir, name, string(zstdBytes(t, want)))
			f := NewFetcher(&Restrictions{AllowedDirs: []string{dir}})
			f.CacheDir = t.TempDir()
			got, err := readAllSource(t, f, mustParse(t, p))
			if err != nil {
				t.Fatalf("reading a local %s: %v", name, err)
			}
			if got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

// TestOpenLocalGzipDecompressedCap: the decompressed-size cap has to hold on
// the streaming local path too, where the bytes never land in a temp file that
// writeTempCounted could count. A few hundred compressed bytes expand past the
// cap, and that must surface as an error rather than a short read.
func TestOpenLocalGzipDecompressedCap(t *testing.T) {
	dir := t.TempDir()
	bomb := gzipBytes(t, strings.Repeat("a", 200_000))
	if len(bomb) > 1000 {
		t.Fatalf("fixture is %d compressed bytes, expected well under the 1000-byte cap", len(bomb))
	}
	p := writeTempFile(t, dir, "bomb.csv.gz", string(bomb))

	f := NewFetcher(&Restrictions{AllowedDirs: []string{dir}})
	f.CacheDir = t.TempDir()
	f.MaxBytes = 1000
	got, err := readAllSource(t, f, mustParse(t, p))
	if err == nil {
		t.Fatalf("expected the decompressed-size cap to trigger, read %d bytes", len(got))
	}
	if !strings.Contains(err.Error(), "decompressed") {
		t.Fatalf("got %v, want a decompressed-size error", err)
	}
	if int64(len(got)) > f.MaxBytes {
		t.Fatalf("the reader handed out %d bytes, more than the %d-byte cap", len(got), f.MaxBytes)
	}
}

// TestOpenLocalGzipExactlyAtCap: the cap is inclusive. Content of exactly
// MaxBytes bytes is legal, so an off-by-one in the streaming counter would
// reject a file the temp-file path accepts.
func TestOpenLocalGzipExactlyAtCap(t *testing.T) {
	want := strings.Repeat("a", 1000)
	dir := t.TempDir()
	p := writeTempFile(t, dir, "exact.csv.gz", string(gzipBytes(t, want)))

	f := NewFetcher(&Restrictions{AllowedDirs: []string{dir}})
	f.CacheDir = t.TempDir()
	f.MaxBytes = int64(len(want))
	got, err := readAllSource(t, f, mustParse(t, p))
	if err != nil {
		t.Fatalf("content of exactly MaxBytes bytes was rejected: %v", err)
	}
	if got != want {
		t.Fatalf("got %d bytes, want %d", len(got), len(want))
	}
}

// TestOpenLocalZstdDecompressedCap: same guard for zstd, whose frames expand
// even more aggressively than gzip's.
func TestOpenLocalZstdDecompressedCap(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "bomb.csv.zst", string(zstdBytes(t, strings.Repeat("a", 200_000))))

	f := NewFetcher(&Restrictions{AllowedDirs: []string{dir}})
	f.CacheDir = t.TempDir()
	f.MaxBytes = 1000
	_, err := readAllSource(t, f, mustParse(t, p))
	if err == nil {
		t.Fatalf("expected the decompressed-size cap to trigger")
	}
	if !strings.Contains(err.Error(), "decompressed") {
		t.Fatalf("got %v, want a decompressed-size error", err)
	}
}

// TestOpenMmapLocalGzipCachesDecompression: a mapping needs a real file, so a
// compressed local source is decompressed into the cache directory. The second
// mapping of an unchanged file must reuse that entry instead of decompressing
// again — asserted on the entry's mtime, which a rewrite would move.
func TestOpenMmapLocalGzipCachesDecompression(t *testing.T) {
	want := "col\nvalue\n"
	dir := t.TempDir()
	cacheDir := t.TempDir()
	p := writeTempFile(t, dir, "data.csv.gz", string(gzipBytes(t, want)))

	f := NewFetcher(&Restrictions{AllowedDirs: []string{dir}})
	f.CacheDir = cacheDir
	s := mustParse(t, p)

	m1, err := f.OpenMmap(s, time.Hour)
	if err != nil {
		t.Fatalf("OpenMmap (1): %v", err)
	}
	if string(m1) != want {
		t.Fatalf("got %q, want %q", m1, want)
	}
	m1.Unmap()

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d cache entries, want exactly the decompressed file", len(entries))
	}
	before, err := os.Stat(filepath.Join(cacheDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	// A re-decompression would rewrite the entry, moving its mtime forward.
	time.Sleep(20 * time.Millisecond)
	m2, err := f.OpenMmap(s, time.Hour)
	if err != nil {
		t.Fatalf("OpenMmap (2): %v", err)
	}
	if string(m2) != want {
		t.Fatalf("got %q, want %q", m2, want)
	}
	m2.Unmap()

	entries, err = os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d cache entries after the second mapping, want 1", len(entries))
	}
	after, err := os.Stat(filepath.Join(cacheDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("the cached decompression was rebuilt: mtime moved from %v to %v", before.ModTime(), after.ModTime())
	}
}

// TestOpenMmapLocalGzipInvalidatedOnChange is the other half of the caching
// contract: the key covers the source file's mtime and size, so editing the
// file must miss the cache rather than serve the previous decompression. The
// two rewrites separate the two components — the first changes the size, the
// second keeps it identical so only the mtime can distinguish them.
func TestOpenMmapLocalGzipInvalidatedOnChange(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "data.csv.gz")

	f := NewFetcher(&Restrictions{AllowedDirs: []string{dir}})
	f.CacheDir = t.TempDir()
	s := mustParse(t, p)

	mapped := func(step string) string {
		t.Helper()
		m, err := f.OpenMmap(s, time.Hour)
		if err != nil {
			t.Fatalf("OpenMmap (%s): %v", step, err)
		}
		defer m.Unmap()
		return string(m)
	}

	writeTempFile(t, dir, "data.csv.gz", string(gzipBytes(t, "first\n")))
	if got := mapped("first"); got != "first\n" {
		t.Fatalf("got %q, want %q", got, "first\n")
	}

	// Different length, so the size component alone differs.
	writeTempFile(t, dir, "data.csv.gz", string(gzipBytes(t, "second value\n")))
	if got := mapped("resized"); got != "second value\n" {
		t.Fatalf("a resized source served the cached decompression: %q", got)
	}

	// Same compressed length, so only the mtime can tell the two apart. It is
	// set explicitly rather than relying on the filesystem's timestamp
	// granularity to make the rewrite observable.
	before, err := os.Stat(p)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	sameLength := gzipBytes(t, "third_ value\n")
	writeTempFile(t, dir, "data.csv.gz", string(sameLength))
	after, err := os.Stat(p)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if after.Size() != before.Size() {
		t.Fatalf("fixture sizes differ (%d vs %d); this case must isolate the mtime component",
			before.Size(), after.Size())
	}
	touched := before.ModTime().Add(time.Second)
	if err := os.Chtimes(p, touched, touched); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	if got := mapped("touched"); got != "third_ value\n" {
		t.Fatalf("a same-size source with a newer mtime served the cached decompression: %q", got)
	}
}

// TestOpenMmapLocalGzipRespectsSandbox: the decompress-to-cache detour must go
// through OpenLocal, or it would be a way to read a file the policy forbids.
func TestOpenMmapLocalGzipRespectsSandbox(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "data.csv.gz", string(gzipBytes(t, "col\nvalue\n")))

	f := NewFetcher(&Restrictions{AllowedDirs: []string{t.TempDir()}})
	f.CacheDir = t.TempDir()
	if _, err := f.OpenMmap(mustParse(t, p), time.Hour); err == nil {
		t.Fatalf("a .gz outside the allowed directories was mapped")
	}
}

// TestFetchHTTPZstdContent: a remote .zst is decoded on the way into the cache,
// so the cached entry (and every reader) holds plain content.
func TestFetchHTTPZstdContent(t *testing.T) {
	want := "a,b\n1,2\n"
	payload := zstdBytes(t, want)

	mux := http.NewServeMux()
	mux.HandleFunc("/data.csv.zst", func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	})
	// The same bytes served without a telltale extension, announced by header.
	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "zstd")
		w.Write(payload)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/data.csv.zst", "/data"} {
		t.Run(path, func(t *testing.T) {
			f := NewFetcher(nil)
			f.CacheDir = t.TempDir()
			got, err := readAllSource(t, f, mustParse(t, srv.URL+path))
			if err != nil {
				t.Fatalf("fetching %s: %v", path, err)
			}
			if got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

// TestFetchHTTPZstdDecompressedCap: the independent decompressed-size cap
// applies to zstd exactly as it does to gzip.
func TestFetchHTTPZstdDecompressedCap(t *testing.T) {
	payload := zstdBytes(t, strings.Repeat("a", 200_000))
	mux := http.NewServeMux()
	mux.HandleFunc("/bomb.zst", func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	f.MaxBytes = 1000
	_, err := f.Open(mustParse(t, srv.URL+"/bomb.zst"), time.Hour)
	if err == nil {
		t.Fatalf("expected the decompressed-size cap to trigger")
	}
	if !strings.Contains(err.Error(), "decompressed") {
		t.Fatalf("got %v, want a decompressed-size error", err)
	}
}

// TestFetchHTTPRevalidatesWithETag: a stale entry is revalidated, not blindly
// re-downloaded. The counters distinguish the three cases that matter — a first
// unconditional 200, a conditional GET answered 304 whose body is reused, and a
// still-fresh entry that must not reach the server at all.
func TestFetchHTTPRevalidatesWithETag(t *testing.T) {
	const etag = `"v1"`
	const body = "body-v1"
	var requests, conditional int32

	mux := http.NewServeMux()
	mux.HandleFunc("/f.txt", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			atomic.AddInt32(&conditional, 1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	s := mustParse(t, srv.URL+"/f.txt")

	read := func(step string, ttl time.Duration) string {
		t.Helper()
		rc, err := f.Open(s, ttl)
		if err != nil {
			t.Fatalf("Open (%s): %v", step, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("ReadAll (%s): %v", step, err)
		}
		return string(data)
	}

	if got := read("first", time.Hour); got != body {
		t.Fatalf("first fetch got %q, want %q", got, body)
	}
	if n := atomic.LoadInt32(&requests); n != 1 {
		t.Fatalf("first fetch made %d requests, want 1", n)
	}

	time.Sleep(10 * time.Millisecond)
	if got := read("revalidated", time.Millisecond); got != body {
		t.Fatalf("revalidated fetch got %q, want %q", got, body)
	}
	if n := atomic.LoadInt32(&requests); n != 2 {
		t.Fatalf("revalidation made %d total requests, want 2", n)
	}
	if n := atomic.LoadInt32(&conditional); n != 1 {
		t.Fatalf("the stale entry was refetched instead of revalidated (%d conditional requests)", n)
	}

	if got := read("fresh", time.Hour); got != body {
		t.Fatalf("fresh fetch got %q, want %q", got, body)
	}
	if n := atomic.LoadInt32(&requests); n != 2 {
		t.Fatalf("a fresh entry hit the network: %d total requests, want 2", n)
	}
}

// TestFetchHTTPRevalidatesWithLastModified covers the servers that offer no
// entity tag: If-Modified-Since must be sent with the exact Last-Modified
// string the previous 200 returned, never a date derived from the cache file.
func TestFetchHTTPRevalidatesWithLastModified(t *testing.T) {
	lastModified := time.Now().UTC().Add(-time.Hour).Format(http.TimeFormat)
	const body = "body-v1"
	var conditional int32

	mux := http.NewServeMux()
	mux.HandleFunc("/f.txt", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-Modified-Since"); got != "" {
			if got != lastModified {
				t.Errorf("If-Modified-Since = %q, want the echoed %q", got, lastModified)
			}
			atomic.AddInt32(&conditional, 1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Last-Modified", lastModified)
		w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	s := mustParse(t, srv.URL+"/f.txt")

	if got, err := readAllSource(t, f, s); err != nil || got != body {
		t.Fatalf("first fetch: %q, %v", got, err)
	}

	time.Sleep(10 * time.Millisecond)
	rc, err := f.Open(s, time.Millisecond)
	if err != nil {
		t.Fatalf("Open (revalidated): %v", err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll (revalidated): %v", err)
	}
	if string(data) != body {
		t.Fatalf("revalidated fetch got %q, want %q", data, body)
	}
	if n := atomic.LoadInt32(&conditional); n != 1 {
		t.Fatalf("got %d conditional requests, want 1", n)
	}
}

// TestFetchHTTPRefetchesWhenChanged: revalidation must not pin a stale body.
// When the validator no longer matches, the 200 replaces both the body and its
// sidecar, so the next revalidation uses the new entity tag.
func TestFetchHTTPRefetchesWhenChanged(t *testing.T) {
	var version int32
	mux := http.NewServeMux()
	mux.HandleFunc("/f.txt", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&version, 1)
		etag := fmt.Sprintf("\"v%d\"", n)
		w.Header().Set("ETag", etag)
		fmt.Fprintf(w, "version-%d", n)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(nil)
	f.CacheDir = t.TempDir()
	s := mustParse(t, srv.URL+"/f.txt")

	first, err := readAllSource(t, f, s)
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	rc, err := f.Open(s, time.Millisecond)
	if err != nil {
		t.Fatalf("Open (2): %v", err)
	}
	second, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll (2): %v", err)
	}
	if string(second) == first {
		t.Fatalf("a changed resource was served from the stale cache entry: %q", first)
	}
}

// TestOpenMmapStdin: stdin has no random access of its own, so it is spooled to
// a file in the cache directory and mapped from there; the spool file must not
// be left behind.
func TestOpenMmapStdin(t *testing.T) {
	want := "spooled stdin content"
	cacheDir := t.TempDir()

	f := NewFetcher(nil)
	f.CacheDir = cacheDir
	pipeStdin(t, []byte(want))

	m, err := f.OpenMmap(mustParse(t, "stdin"), time.Hour)
	if err != nil {
		t.Fatalf("OpenMmap(stdin): %v", err)
	}
	if string(m) != want {
		t.Fatalf("got %q, want %q", m, want)
	}
	m.Unmap()

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("the stdin spool file was left in the cache dir: %d entries", len(entries))
	}
}

// TestOpenMmapStdinDeniedUnderSandbox: mapping stdin goes through the same gate
// as reading it, so a non-nil policy still denies it.
func TestOpenMmapStdinDeniedUnderSandbox(t *testing.T) {
	f := NewFetcher(&Restrictions{})
	f.CacheDir = t.TempDir()
	if _, err := f.OpenMmap(mustParse(t, "stdin"), time.Hour); err == nil {
		t.Fatalf("stdin was memory-mapped under a sandbox")
	}
}

func TestParseByteSize(t *testing.T) {
	ok := []struct {
		in   string
		want int64
	}{
		{"1024", 1024},
		{"  2048  ", 2048},
		{"512B", 512},
		{"32GiB", 32 << 30},
		{"64 gib", 64 << 30},
		{"1KB", 1000},
		{"2MB", 2_000_000},
		{"3GB", 3_000_000_000},
		{"1TB", 1_000_000_000_000},
		{"8K", 8 << 10},
		{"4M", 4 << 20},
		{"2G", 2 << 30},
		{"1T", 1 << 40},
	}
	for _, tc := range ok {
		got, err := parseByteSize(tc.in)
		if err != nil {
			t.Errorf("parseByteSize(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseByteSize(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}

	bad := []string{"", "abc", "1.5GB", "-1", "0", "12PB", "GB", "9223372036854775807TiB"}
	for _, in := range bad {
		if got, err := parseByteSize(in); err == nil {
			t.Errorf("parseByteSize(%q) = %d, want an error", in, got)
		}
	}
}

func TestResolveMaxDownloadSize(t *testing.T) {
	if got := resolveMaxDownloadSize("512MB"); got != 512_000_000 {
		t.Errorf("resolveMaxDownloadSize(\"512MB\") = %d, want 512000000", got)
	}
	if got := resolveMaxDownloadSize(" 64GiB "); got != 64<<30 {
		t.Errorf("resolveMaxDownloadSize(\"64GiB\") = %d, want %d", got, int64(64)<<30)
	}
	// An unset or unusable value must read as "nobody configured a cap", so the
	// defaults apply. Reporting a cap of 0 would make every fetch unbounded.
	for _, in := range []string{"", "   ", "banana", "-5", "0"} {
		if got := resolveMaxDownloadSize(in); got != 0 {
			t.Errorf("resolveMaxDownloadSize(%q) = %d, want 0", in, got)
		}
	}
}

// One cap governs every bounded path, so a compressed file is readable at the
// same size its plain form is. An unset environment leaves the default in
// place: resolving to 0 would make the fetch unbounded.
func TestMaxBytesDefaultsAndOverride(t *testing.T) {
	f := NewFetcher(nil)
	if got := f.maxBytes(); got != defaultMaxDownloadSize {
		t.Fatalf("maxBytes() = %d, want %d", got, defaultMaxDownloadSize)
	}
	f.MaxBytes = 1234
	if got := f.maxBytes(); got != 1234 {
		t.Fatalf("maxBytes() = %d, want the explicit 1234", got)
	}
}
