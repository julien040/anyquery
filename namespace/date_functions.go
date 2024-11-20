package namespace

import (
	"fmt"
	"time"

	"github.com/GuilhermeCaruso/kair"
	"github.com/araddon/dateparse"
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
		{"toDateFormatted", toDateFormatted, true},
		{"toDateFormatted", toDateFormattedInt64, true},
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
	// "07 oct 2014", "1732086477"
	t, _ := dateparse.ParseAny(date)

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

func toDateFormatted(userVal string, format string) string {
	parsed := parseDate(userVal)
	skair := kair.DateTime(parsed.Day(), int(parsed.Month()), parsed.Year(), parsed.Hour(), parsed.Minute(), parsed.Second())
	return skair.CustomFormat(format)
}

func toDateFormattedInt64(userVal int64, format string) string {
	parsed := parseDate(fmt.Sprintf("%d", userVal))
	skair := kair.DateTime(parsed.Day(), int(parsed.Month()), parsed.Year(), parsed.Hour(), parsed.Minute(), parsed.Second())
	return skair.CustomFormat(format)
}
