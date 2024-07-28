package namespace

import (
	"net/url"

	"github.com/mattn/go-sqlite3"
)

// This file defines the url functions that are available in SQL queries
//
// If the function has multiple alias, please specify the different names in the comment

func registerURLFunctions(conn *sqlite3.SQLiteConn) {
	var urlFunctions = []struct {
		name     string
		function any
		pure     bool
	}{
		{"url_encode", encodeURLComponent, true},
		{"urlEncode", encodeURLComponent, true},
		{"url_decode", decodeURLComponent, true},
		{"urlDecode", decodeURLComponent, true},
		{"url_domain", domain, true},
		{"urlDomain", domain, true},
		{"domain", domain, true},
		{"urlPort", port, true},
		{"port", port, true},
		{"urlPath", path, true},
		{"path", path, true},
		{"url_path", path, true}, // Alias: urlPath
		{"urlQuery", query, true},
		{"url_query", query, true}, // Alias: urlQuery
		{"urlParameter", extractURLParameter, true},
		{"url_parameter", extractURLParameter, true},         // Alias: urlParameter
		{"extractURLParameter", extractURLParameter, true},   // Alias: extractURLParameter
		{"extract_url_parameter", extractURLParameter, true}, // Alias: extractURLParameter
		{"urlProtocol", protocol, true},
		{"protocol", protocol, true},
		{"url_protocol", protocol, true}, // Alias: urlProtocol
	}
	for _, f := range urlFunctions {
		conn.RegisterFunc(f.name, f.function, f.pure)
	}
}

func domain(urlstr string) string {
	parsed, err := url.Parse(urlstr)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func port(urlstr string) string {
	parsed, err := url.Parse(urlstr)
	if err != nil {
		return ""
	}
	return parsed.Port()
}

func path(urlstr string) string {
	parsed, err := url.Parse(urlstr)
	if err != nil {
		return ""
	}
	return parsed.Path
}

func query(urlstr string) string {
	parsed, err := url.Parse(urlstr)
	if err != nil {
		return ""
	}
	return parsed.RawQuery
}

func encodeURLComponent(str string) string {
	return url.QueryEscape(str)
}

func decodeURLComponent(str string) string {
	decoded, err := url.QueryUnescape(str)
	if err != nil {
		return ""
	}
	return decoded
}

func extractURLParameter(urlstr, key string) string {
	parsed, err := url.Parse(urlstr)
	if err != nil {
		return ""
	}
	return parsed.Query().Get(key)
}

func protocol(urlstr string) string {
	parsed, err := url.Parse(urlstr)
	if err != nil {
		return ""
	}
	return parsed.Scheme
}
