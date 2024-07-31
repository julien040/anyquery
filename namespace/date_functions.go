package namespace

import (
	"time"

	"github.com/mattn/go-sqlite3"
)

func registerDateFunctions(conn *sqlite3.SQLiteConn) {
	var dateFunctions = []struct {
		name     string
		function any
		pure     bool
	}{
		{"utc_timestamp", utc_timestamp, false},
		{"now", now, false},
		{"toYYYYMMDDHHMMSS", toYYYYMMDDHHMMSS, true},
		{"toYYYYMMDD", toYYYYMMDD, true},
		{"toYYYYMM", toYYYYMM, true},
		{"toYYYY", toYYYY, true},
		{"toHH", toHH, true},
		{"toMM", toMM, true},
		{"toSS", toSS, true},
	}
	for _, f := range dateFunctions {
		conn.RegisterFunc(f.name, f.function, f.pure)
	}
}

func utc_timestamp() string {
	utc := time.Now().UTC()
	return utc.Format("2006-01-02 15:04:05")
}

func now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func parseDate(date string) time.Time {
	// Try to parse in different formats
	// such as "YYYY-MM-DD", "YYYY-MM-DD HH:MM:SS", "YYYY-MM-DD HH:MM:SS.ssssss"

	formats := []string{

		time.RFC1123,
		time.RFC1123Z,
		time.UnixDate,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000000",
		"15:04:05",
		"15:04:05.000000",
	}

	var t time.Time
	var err error
	for _, format := range formats {
		t, err = time.Parse(format, date)
		if err == nil {
			return t
		}
	}

	return t
}

func toYYYYMMDDHHMMSS(userVal string) string {
	parsed := parseDate(userVal)
	return parsed.Format("2006-01-02 15:04:05")
}

func toYYYYMMDD(userVal string) string {
	parsed := parseDate(userVal)
	return parsed.Format("2006-01-02")
}

func toYYYYMM(userVal string) string {
	parsed := parseDate(userVal)
	return parsed.Format("2006-01")
}

func toYYYY(userVal string) string {
	parsed := parseDate(userVal)
	return parsed.Format("2006")
}

func toHH(userVal string) string {
	parsed := parseDate(userVal)
	return parsed.Format("15")
}

func toMM(userVal string) string {
	parsed := parseDate(userVal)
	return parsed.Format("04")
}

func toSS(userVal string) string {
	parsed := parseDate(userVal)
	return parsed.Format("05")
}
