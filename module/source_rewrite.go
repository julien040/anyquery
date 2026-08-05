package module

import (
	"fmt"
	"net/url"
	"strings"
)

// rewriteAndClassify applies the host rewrite table to an http(s) URL and
// classifies the result. Placement matters: this runs inside ParseSource, so
// the rewritten URL is what gets both policy-checked and fetched — applying
// the same transformation after the policy check would let the checked URL
// differ from the fetched one. Host matching is exact, case-insensitive equality on
// u.Hostname(); never HasSuffix/Contains, which would let
// "github.com.evil.tld", "evil-github.com" or "notgist.github.com" match.
func rewriteAndClassify(raw string, u *url.URL) (Source, error) {
	if u.User != nil {
		if _, closed := rewriteHost(u.Hostname()); closed {
			return Source{}, fmt.Errorf("source: %q may not contain userinfo", raw)
		}
	}
	switch strings.ToLower(u.Hostname()) {
	case "github.com", "www.github.com":
		return rewriteGitHub(raw, u)
	case "gist.github.com":
		return rewriteGist(raw, u)
	case "huggingface.co":
		return rewriteHFBlob(raw, u)
	case "docs.google.com":
		return rewriteSheets(raw, u)
	case "www.dropbox.com", "dropbox.com":
		return rewriteDropbox(raw, u)
	case "gitlab.com", "www.gitlab.com":
		return rewriteGitLab(raw, u)
	case "codeberg.org":
		return rewriteCodeberg(raw, u)
	default:
		return Source{Kind: KindHTTP, URL: u, Raw: raw}, nil
	}
}

// rewriteHost reports whether host is one of the closed set of rewrite hosts.
func rewriteHost(host string) (name string, closed bool) {
	switch strings.ToLower(host) {
	case "github.com", "www.github.com":
		return "github", true
	case "gist.github.com":
		return "gist", true
	case "huggingface.co":
		return "huggingface", true
	case "docs.google.com":
		return "google sheets", true
	case "www.dropbox.com", "dropbox.com":
		return "dropbox", true
	case "gitlab.com", "www.gitlab.com":
		return "gitlab", true
	case "codeberg.org":
		return "codeberg", true
	default:
		return "", false
	}
}

func pathSegments(u *url.URL) []string {
	p := strings.Trim(u.Path, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// rewriteGitHub rewrites a github.com / www.github.com file URL to its
// raw.githubusercontent.com raw-content form.
func rewriteGitHub(raw string, u *url.URL) (Source, error) {
	segs := pathSegments(u)
	if len(segs) < 3 {
		return Source{}, fmt.Errorf("source: %q does not identify a file (expected /{owner}/{repo}/blob/{ref}/{path})", raw)
	}
	owner, repo, kind := segs[0], segs[1], segs[2]
	switch kind {
	case "blob", "raw":
		tail := strings.Join(segs[3:], "/")
		if tail == "" {
			return Source{}, fmt.Errorf("source: %q does not identify a file (expected /{owner}/{repo}/blob/{ref}/{path})", raw)
		}
		newURL := &url.URL{Scheme: "https", Host: "raw.githubusercontent.com", Path: "/" + owner + "/" + repo + "/" + tail}
		return Source{Kind: KindHTTP, URL: newURL, Raw: raw, RewriteNote: raw + " → " + newURL.String()}, nil
	case "tree":
		return Source{}, fmt.Errorf("source: %q is a directory, not a file", raw)
	default:
		return Source{}, fmt.Errorf("source: %q does not identify a file (expected /{owner}/{repo}/blob/{ref}/{path})", raw)
	}
}

// rewriteGist rewrites a gist.github.com URL to its
// gist.githubusercontent.com raw form.
func rewriteGist(raw string, u *url.URL) (Source, error) {
	segs := pathSegments(u)
	if len(segs) < 2 {
		return Source{}, fmt.Errorf("source: %q is missing the owner segment (expected /{owner}/{gist-id})", raw)
	}
	owner, id := segs[0], segs[1]
	var path string
	switch {
	case len(segs) == 2:
		path = "/" + owner + "/" + id + "/raw"
	case segs[2] == "raw":
		tail := strings.Join(segs[3:], "/")
		if tail == "" {
			path = "/" + owner + "/" + id + "/raw"
		} else {
			path = "/" + owner + "/" + id + "/raw/" + tail
		}
	default:
		return Source{}, fmt.Errorf("source: %q is not a supported gist URL", raw)
	}
	newURL := &url.URL{Scheme: "https", Host: "gist.githubusercontent.com", Path: path}
	return Source{Kind: KindHTTP, URL: newURL, Raw: raw, RewriteNote: raw + " → " + newURL.String()}, nil
}

// rewriteGitLab rewrites a gitlab.com project file URL to its raw form:
// "/{group}/{project}/-/blob/{ref}/{path}" becomes
// "/{group}/{project}/-/raw/{ref}/{path}". GitLab puts a literal "/-/"
// separator between the project path and the resource, and the project path may
// be nested in subgroups ("/group/sub/project/-/blob/…"), so the split is on
// the first "/-/" rather than on a fixed segment count. The tail after "blob/"
// is passed through verbatim because a ref may itself contain slashes
// ("feature/my-branch"). The rewrite stays on gitlab.com — only the path
// changes — and still reports a RewriteNote, since the fetched path is not the
// one the query named.
func rewriteGitLab(raw string, u *url.URL) (Source, error) {
	path := strings.Trim(u.Path, "/")
	i := strings.Index(path, "/-/")
	if i < 0 {
		return Source{}, errGitLabForm(raw)
	}
	project, rest := path[:i], path[i+len("/-/"):]
	projectSegs := strings.Split(project, "/")
	if len(projectSegs) < 2 {
		return Source{}, errGitLabForm(raw)
	}
	for _, seg := range projectSegs {
		if seg == "" {
			return Source{}, errGitLabForm(raw)
		}
	}
	restSegs := strings.Split(rest, "/")
	switch restSegs[0] {
	case "blob":
		tail := strings.Join(restSegs[1:], "/")
		if tail == "" {
			return Source{}, errGitLabForm(raw)
		}
		// The input's query/fragment are dropped: GitLab appends viewer
		// parameters such as "?ref_type=heads" that mean nothing to the raw
		// endpoint.
		newURL := &url.URL{Scheme: "https", Host: "gitlab.com", Path: "/" + project + "/-/raw/" + tail}
		return Source{Kind: KindHTTP, URL: newURL, Raw: raw, RewriteNote: raw + " → " + newURL.String()}, nil
	case "raw":
		// Already the raw endpoint on the same host: nothing to rewrite.
		return Source{Kind: KindHTTP, URL: u, Raw: raw}, nil
	case "tree":
		return Source{}, fmt.Errorf("source: %q is a directory, not a file", raw)
	default:
		return Source{}, errGitLabForm(raw)
	}
}

func errGitLabForm(raw string) error {
	return fmt.Errorf("source: %q does not identify a file (expected /{group}/{project}/-/blob/{ref}/{path})", raw)
}

// rewriteCodeberg rewrites a codeberg.org (Forgejo/Gitea layout) file URL to
// its raw form: "/{owner}/{repo}/src/{branch|tag|commit}/{ref}/{path}" becomes
// "/{owner}/{repo}/raw/{branch|tag|commit}/{ref}/{path}". The tail after the
// selector is passed through verbatim because a ref may itself contain slashes
// ("feature/my-branch"); that also means a ref cannot be told apart from a ref
// plus a file by counting, so at least two tail segments are required — a
// ".../src/branch/{ref}" URL with nothing after the ref is a branch listing,
// and its raw counterpart could never be a file. The rewrite stays on
// codeberg.org — only the path changes — and still reports a RewriteNote,
// since the fetched path is not the one the query named.
func rewriteCodeberg(raw string, u *url.URL) (Source, error) {
	segs := pathSegments(u)
	if len(segs) < 3 || segs[0] == "" || segs[1] == "" {
		return Source{}, errCodebergForm(raw)
	}
	owner, repo := segs[0], segs[1]
	switch segs[2] {
	case "src":
		if len(segs) < 6 {
			return Source{}, errCodebergForm(raw)
		}
		selector := segs[3]
		if selector != "branch" && selector != "tag" && selector != "commit" {
			return Source{}, errCodebergForm(raw)
		}
		tail := strings.Join(segs[4:], "/")
		// The input's query/fragment are dropped: they are viewer parameters
		// (e.g. "?display=source") the raw endpoint has no use for.
		newURL := &url.URL{Scheme: "https", Host: "codeberg.org", Path: "/" + owner + "/" + repo + "/raw/" + selector + "/" + tail}
		return Source{Kind: KindHTTP, URL: newURL, Raw: raw, RewriteNote: raw + " → " + newURL.String()}, nil
	case "raw":
		// Already the raw endpoint on the same host: nothing to rewrite.
		return Source{Kind: KindHTTP, URL: u, Raw: raw}, nil
	default:
		return Source{}, errCodebergForm(raw)
	}
}

func errCodebergForm(raw string) error {
	return fmt.Errorf("source: %q does not identify a file (expected /{owner}/{repo}/src/branch/{ref}/{path}, or src/tag/… or src/commit/…)", raw)
}

// rewriteHFBlob rewrites a huggingface.co dataset "/blob/" (HTML page) URL to
// its "/resolve/" (raw content) form. Any other
// huggingface.co URL (e.g. one already in /resolve/ form) passes through
// unchanged.
func rewriteHFBlob(raw string, u *url.URL) (Source, error) {
	segs := pathSegments(u)
	if len(segs) >= 4 && segs[0] == "datasets" && segs[3] == "blob" {
		tail := strings.Join(segs[4:], "/")
		if tail == "" {
			return Source{}, fmt.Errorf("source: %q does not identify a file", raw)
		}
		newURL := &url.URL{Scheme: "https", Host: "huggingface.co", Path: "/datasets/" + segs[1] + "/" + segs[2] + "/resolve/" + tail}
		return Source{Kind: KindHTTP, URL: newURL, Raw: raw, RewriteNote: raw + " → " + newURL.String()}, nil
	}
	return Source{Kind: KindHTTP, URL: u, Raw: raw}, nil
}

// rewriteHuggingFace converts an "hf://" source into the equivalent https URL
// for rewriteAndClassify to finish classifying.
// url.Parse splits "hf://datasets/{u}/{d}/{path}" as Host="datasets",
// Path="/{u}/{d}/{path}" (there is no real authority component in this
// scheme), so the segments are reassembled from Host+Path rather than Path
// alone.
func rewriteHuggingFace(raw string, u *url.URL) (*url.URL, error) {
	if u.User != nil {
		return nil, fmt.Errorf("source: %q may not contain userinfo", raw)
	}
	var segs []string
	if u.Host != "" {
		segs = append(segs, u.Host)
	}
	segs = append(segs, pathSegments(u)...)
	if len(segs) == 0 {
		return nil, fmt.Errorf("source: %q does not identify a file", raw)
	}
	if segs[0] != "datasets" {
		return nil, fmt.Errorf("source: %q is unsupported; only hf://datasets/… is supported", raw)
	}
	if len(segs) < 3 {
		return nil, fmt.Errorf("source: %q is missing the dataset identifier (expected hf://datasets/{user}/{dataset}/{path})", raw)
	}
	user, datasetAndRev := segs[1], segs[2]
	dataset, rev := datasetAndRev, "main"
	if i := strings.Index(datasetAndRev, "@"); i >= 0 {
		dataset, rev = datasetAndRev[:i], datasetAndRev[i+1:]
	}
	tail := strings.Join(segs[3:], "/")
	if tail == "" {
		return nil, fmt.Errorf("source: %q does not identify a file", raw)
	}
	if strings.ContainsAny(tail, "*?") {
		return nil, fmt.Errorf("source: %q uses a glob, which is unsupported; hf:// sources must name a single file", raw)
	}
	if dataset == "" || rev == "" {
		return nil, fmt.Errorf("source: %q has an empty dataset or revision", raw)
	}
	newURL := &url.URL{Scheme: "https", Host: "huggingface.co", Path: "/datasets/" + user + "/" + dataset + "/resolve/" + rev + "/" + tail}
	return newURL, nil
}

// rewriteSheets rewrites a Google Sheets share/edit URL to the sheet's CSV
// export form.
func rewriteSheets(raw string, u *url.URL) (Source, error) {
	segs := pathSegments(u)
	if len(segs) < 3 || segs[0] != "spreadsheets" || segs[1] != "d" || segs[2] == "" {
		return Source{}, fmt.Errorf("source: %q is not a supported Google Sheets URL", raw)
	}
	id := segs[2]
	if len(segs) == 3 {
		return buildSheetsSource(raw, id, "")
	}
	switch segs[3] {
	case "edit":
		gid := ""
		if u.Fragment != "" {
			if !strings.HasPrefix(u.Fragment, "gid=") {
				return Source{}, fmt.Errorf("source: %q has an unsupported fragment", raw)
			}
			gid = strings.TrimPrefix(u.Fragment, "gid=")
			if !isDigits(gid) {
				return Source{}, fmt.Errorf("source: %q has a non-numeric gid", raw)
			}
		}
		return buildSheetsSource(raw, id, gid)
	case "export":
		return Source{Kind: KindHTTP, URL: u, Raw: raw}, nil
	default:
		return Source{}, fmt.Errorf("source: %q is not a supported Google Sheets URL", raw)
	}
}

func buildSheetsSource(raw, id, gid string) (Source, error) {
	q := url.Values{"format": {"csv"}}
	if gid != "" {
		q.Set("gid", gid)
	}
	newURL := &url.URL{Scheme: "https", Host: "docs.google.com", Path: "/spreadsheets/d/" + id + "/export", RawQuery: q.Encode()}
	return Source{Kind: KindHTTP, URL: newURL, Raw: raw, RewriteNote: raw + " → " + newURL.String()}, nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// rewriteDropbox rewrites a Dropbox share link to fetch the raw file (raw=1).
func rewriteDropbox(raw string, u *url.URL) (Source, error) {
	q := u.Query()
	q.Del("dl")
	q.Set("raw", "1")
	newURL := *u
	newURL.RawQuery = q.Encode()
	return Source{Kind: KindHTTP, URL: &newURL, Raw: raw, RewriteNote: raw + " → " + newURL.String()}, nil
}
