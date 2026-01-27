package namespace

import (
	"github.com/Masterminds/semver/v3"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
)

func registerCollations(conn *sqlite3.SQLiteConn) {
	var collations = []struct {
		name string
		cmp  func(string, string) int
	}{
		{"semver", semverCollation},
	}

	for _, c := range collations {
		conn.RegisterCollation(c.name, c.cmp)
	}
}

func semverCollation(a, b string) int {
	semverA, err := semver.NewVersion(a)
	if err != nil {
		return -1
	}

	semverB, err := semver.NewVersion(b)
	if err != nil {
		return 1
	}

	return semverA.Compare(semverB)
}
