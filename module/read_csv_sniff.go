package module

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
)

// The sniffer looks at the head of the file only, so opening a table stays
// cheap on a large file. 128 KiB holds a few hundred rows of a typical CSV,
// which is plenty to tell a delimiter and a column type apart.
const (
	csvSniffMaxBytes   = 128 * 1024
	csvSniffMaxRecords = 500
)

// csvSniffSeparators are the delimiters tried when the caller did not pass one.
// The order is the tie-breaker: a comma wins a candidate that scores the same.
var csvSniffSeparators = []rune{',', '\t', ';', '|'}

// sniffCSV infers what the caller left unspecified from the head of file: the
// delimiter when sepExplicit is false, the presence of a header row when
// headerSet is false, and always the column names and types.
//
// An explicit value is never second-guessed: sep is used as is when sepExplicit
// is set, and headerVal is used as is when headerSet is set. The returned
// columns are named after the header row when there is one and col0..colN
// otherwise, and carry the colType values CsvCursor.Column converts with
// ("int", "float", "bool" or "string").
//
// A nil column slice means the sample held no record at all, which is the empty
// file the caller has to reject.
func sniffCSV(sample []byte, sep rune, sepExplicit bool, headerSet bool, headerVal bool) (rune, bool, []columnCsv) {
	sample = csvSniffSample(sample)

	var records [][]string
	if sepExplicit {
		records = readCsvSample(sample, sep)
	} else {
		sep, records = detectCsvSeparator(sample)
	}

	if len(records) == 0 {
		return sep, headerSet && headerVal, nil
	}

	hasHeader := headerSet && headerVal
	if !headerSet {
		hasHeader = detectCsvHeader(records)
	}

	// The header row describes the columns, it is not one of their values.
	dataRows := records
	if hasHeader {
		dataRows = records[1:]
	}

	columns := make([]columnCsv, len(records[0]))
	for i := range columns {
		name := fmt.Sprintf("col%d", i)
		if hasHeader {
			name = records[0][i]
		}
		columns[i] = columnCsv{
			name:    name,
			colType: detectCsvColumnType(dataRows, i),
		}
	}

	return sep, hasHeader, columns
}

// csvSniffSample returns the head of file the sniffer works on. When the file is
// longer than the limit, the sample stops at the last line break so that the
// cut does not leave a truncated record behind, which would look like a row
// with fewer fields than the others.
func csvSniffSample(file []byte) []byte {
	if len(file) <= csvSniffMaxBytes {
		return file
	}
	sample := file[:csvSniffMaxBytes]
	if i := bytes.LastIndexByte(sample, '\n'); i >= 0 {
		return sample[:i+1]
	}
	return sample
}

// readCsvSample parses up to csvSniffMaxRecords records of sample with sep.
// Quoting is as lenient as the reader used to serve the table, and records are
// allowed to disagree on their field count because that disagreement is exactly
// what detectCsvSeparator measures. A record that cannot be parsed ends the
// sample: what was read so far is still a usable sample.
func readCsvSample(sample []byte, sep rune) [][]string {
	reader := csv.NewReader(bytes.NewReader(sample))
	reader.Comma = sep
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	records := make([][]string, 0, 16)
	for len(records) < csvSniffMaxRecords {
		record, err := reader.Read()
		if err != nil {
			break
		}
		records = append(records, record)
	}
	return records
}

// detectCsvSeparator picks the delimiter that splits the sample into the most
// columns while keeping the field count identical across records, and returns
// it along with the sample parsed with it.
func detectCsvSeparator(sample []byte) (rune, [][]string) {
	bestSep := ','
	var bestRecords [][]string
	bestScore, bestFields := -1, -1

	for _, sep := range csvSniffSeparators {
		records := readCsvSample(sample, sep)
		if len(records) == 0 {
			continue
		}

		fields := len(records[0])
		consistent := true
		for _, record := range records {
			if len(record) != fields {
				consistent = false
				break
			}
		}

		// A delimiter that splits nothing is only kept when no other one
		// splits the sample either, and an inconsistent split is only kept
		// when no delimiter splits it evenly.
		score := 0
		switch {
		case fields <= 1:
			score = 0
		case consistent:
			score = 2
		default:
			score = 1
		}

		if score > bestScore || (score == bestScore && fields > bestFields) {
			bestSep, bestRecords, bestScore, bestFields = sep, records, score, fields
		}
	}

	if bestRecords == nil {
		return ',', nil
	}
	return bestSep, bestRecords
}

// detectCsvHeader reports whether the first record names the columns instead of
// holding data. It says yes when no cell of the first record parses as a number
// or a boolean while at least one column below it is mostly numeric or boolean,
// so that a file whose rows are all text stays headerless: nothing
// distinguishes its first row, and assuming a header there would silently drop
// a row of data.
func detectCsvHeader(records [][]string) bool {
	if len(records) < 2 {
		return false
	}

	first := records[0]
	named := false
	for _, cell := range first {
		if isCsvTypedValue(cell) {
			return false
		}
		if cell != "" {
			named = true
		}
	}
	if !named {
		return false
	}

	below := records[1:]
	for col := range first {
		typed := 0
		for _, record := range below {
			if col < len(record) && isCsvTypedValue(record[col]) {
				typed++
			}
		}
		if typed*2 > len(below) {
			return true
		}
	}

	return false
}

// detectCsvColumnType returns the narrowest type every value of the column at
// index col parses as. Empty cells carry no type information and are skipped, so
// a column of integers with holes stays an integer column; a column made only of
// them is text.
func detectCsvColumnType(records [][]string, col int) string {
	seen := false
	isInt, isFloat, isBool := true, true, true

	for _, record := range records {
		if col >= len(record) {
			continue
		}
		cell := record[col]
		if cell == "" {
			continue
		}
		seen = true

		if isInt && !isCsvInt(cell) {
			isInt = false
		}
		if isFloat && !isCsvFloat(cell) {
			isFloat = false
		}
		if isBool && !isCsvBool(cell) {
			isBool = false
		}
		if !isInt && !isFloat && !isBool {
			break
		}
	}

	switch {
	case !seen:
		return "string"
	case isInt:
		return "int"
	case isFloat:
		return "float"
	case isBool:
		return "bool"
	default:
		return "string"
	}
}

// isCsvTypedValue reports whether a cell holds something other than free text.
// An empty cell is not typed: it is the same absence of value in every column.
func isCsvTypedValue(cell string) bool {
	if cell == "" {
		return false
	}
	return isCsvInt(cell) || isCsvFloat(cell) || isCsvBool(cell)
}

// The parsing below is the one CsvCursor.Column applies to the raw cell, with
// no trimming and no other leniency. A value accepted here must convert there
// too, otherwise the column would be typed and then read as NULL.

func isCsvInt(cell string) bool {
	_, err := strconv.ParseInt(cell, 10, 64)
	return err == nil
}

func isCsvFloat(cell string) bool {
	if cell == "" {
		return false
	}
	// ParseFloat also accepts "NaN", "Inf" and hexadecimal floats, which are
	// words a text column may well contain. A number starts with a digit, a
	// sign or a decimal point.
	switch cell[0] {
	case '+', '-', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
	default:
		return false
	}
	_, err := strconv.ParseFloat(cell, 64)
	return err == nil
}

func isCsvBool(cell string) bool {
	_, err := strconv.ParseBool(cell)
	return err == nil
}
