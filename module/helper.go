package module

import (
	"regexp"
	"strings"
	"time"

	"github.com/edsrzf/mmap-go"
)

// openMmapedFile parses src (see ParseSource) and returns a mmap of it,
// fetching and caching it first when it is remote. ttl bounds the freshness of
// that cache entry, and therefore only affects a remote source: local sources
// bypass the cache entirely (see module/fetch.go).
func openMmapedFile(src string, r *Restrictions, ttl time.Duration) (mmap.MMap, error) {
	s, err := ParseSource(src)
	if err != nil {
		return nil, err
	}
	return NewFetcher(r).OpenMmap(s, ttl)
}

// To make the argument parsing more readable,
// we define a struct to hold the argument name and its value
type argParam struct {
	name  string
	value *string
}

var argRegExp *regexp.Regexp = regexp.MustCompile("^\\s*['\"`]?([^= '\"`]+?)['\"`]?\\s*=\\s*['\"`]?(.*?)['\"`]?\\s*$")

func parseArgs(params []argParam, args []string) {
	// It's quadratic but the number of arguments is small
	for i := 0; i < len(args); i++ {
		// Check if the argument is empty
		if args[i] == "" {
			continue
		}

		// Parse the argument
		matches := argRegExp.FindStringSubmatch(args[i])
		if matches == nil {
			continue
		}

		matches[1] = strings.ToLower(matches[1])

		// Check if the argument starts with the parameter name
		for j := 0; j < len(params); j++ {
			if matches[1] == params[j].name {
				*params[j].value = matches[2]
				break
			}
		}
	}
}

var sqliteValidName *regexp.Regexp = regexp.MustCompile(`[^\p{L}\p{N}_]+`)

func transformSQLiteValidName(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)
	spaceRemoved := strings.Map(func(r rune) rune {
		if r == ' ' || r == '.' || r == '-' || r == '/' {
			return '_'
		}
		return r
	}, s)

	return sqliteValidName.ReplaceAllString(spaceRemoved, "")
}
