package namespace

import (
	"encoding/base64"
	"encoding/hex"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattn/go-sqlite3"
)

// This file defines the string functions that are available in SQL queries
//
// If the function has multiple alias, please specify the different names in the comment

func registerStringFunctions(conn *sqlite3.SQLiteConn) error {
	functions := []struct {
		Name          string
		Func          interface{}
		Deterministic bool
	}{
		{"ascii", ascii, true},
		{"ord", ascii, true},
		{"bin", bin_num, true},
		{"bin", bin_str, true},
		{"bit_length", bit_length, true},
		{"char", char, true},
		{"chr", char, true},
		{"length", length, true},
		{"char_length", length, true},
		{"character_length", length, true},
		{"elt", elt, true},
		{"elt_word", elt_word, true},
		{"split_part", elt_word, true},
		{"field", field, true},
		{"find_in_set", find_in_set, true},
		{"to_char", to_char, true},
		{"to_char", to_char_float, true},
		{"from_base64", from_base64, true},
		{"from_base64", from_base64_bytes, true},
		{"to_base64", to_base64, true},
		{"to_base64", to_base64_bytes, true},
		{"to_hex", to_hex, true},
		{"to_hex", to_hex_bytes, true},
		{"from_hex", from_hex, true},
		{"from_hex", from_hex_bytes, true},
		{"decode", decode, true},
		{"decode", decode_bytes, true},
		{"encode", encode, true},
		{"encode", encode_bytes, true},
		{"insert", insert, true},
		{"locate", locate, true},
		{"position", locate, true},
		{"locate", locate_from, true},
		{"position", locate_from, true},
		{"lcase", lcase, true},
		{"left", left, true},
		{"load_file", load_file, true},
		{"load_file_bytes", load_file_bytes, true},
		{"lpad", lpad, true},
		{"rpad", rpad, true},
		{"octet_length", octet_length, true},
		{"octet_length", octet_length_bytes, true},
		{"to_octal", to_octal, true},
		{"regexp_replace", regexp_replace, true},
		{"regexp_substr", regexp_substr, true},
		{"repeat", repeat, true},
		{"right", right, true},
		{"reverse", reverse, true},
		{"space", space, true},
		{"ucase", ucase, true},
	}

	for _, f := range functions {
		if err := conn.RegisterFunc(f.Name, f.Func, f.Deterministic); err != nil {
			return err
		}
	}
	return nil
}

// Returns the ascii value of the first character of the input string
// Alias: ord
func ascii(str string) int {
	if str == "" {
		return 0
	}
	return int(str[0])
}

// Returns the binary representation of the number
// Alias: bin
func bin_num(num int) string {
	return strconv.FormatInt(int64(num), 2)
}

// Returns the binary representation of the string
// Alias: bin
func bin_str(str string) []byte {
	return []byte(str)
}

// Returns the length of bits for the string
func bit_length(str string) int {
	return len(str) * 8
}

// Returns the character for the ascii value
// Alias: chr
func char(num ...int8) string {
	byteStr := make([]byte, len(num))
	for i, n := range num {
		byteStr[i] = byte(n)
	}
	return string(byteStr)
}

// Returns the length of the string
// Alias: char_length, character_length, length
func length(str string) int {
	return len(str)
}

// Returns the string at the specified index
func elt(index int, str ...string) string {
	if index <= 0 || index > len(str) {
		return ""
	}
	return str[index-1]
}

// Returns the word at the specified index
// Alias: SPLIT_PART
func elt_word(str string, delim string, index int) string {
	words := strings.Split(str, delim)
	if index <= 0 || index > len(words) {
		return ""
	}
	return words[index-1]
}

// Returns the index of the first string in the argument that matches the pattern
func field(str string, patterns ...string) int {
	for i, pattern := range patterns {
		if str == pattern {
			return i + 1
		}
	}
	return 0
}

// Returns the index in the string list that matches the pattern
func find_in_set(str string, strList string) int {
	strs := strings.Split(strList, ",")
	for i, s := range strs {
		if str == s {
			return i + 1
		}
	}
	return 0
}

func to_char(num int) string {
	return strconv.Itoa(num)
}

// Alias: to_char
func to_char_float(num float64) string {
	return strconv.FormatFloat(num, 'f', -1, 64)
}

// Decode the base64 encoded string
// Alias: from_base64
func from_base64(str string) string {
	bytes, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// Alias: from_base64
func from_base64_bytes(str []byte) string {
	bytes, err := base64.StdEncoding.DecodeString(string(str))
	if err != nil {
		return ""
	}
	return string(bytes)
}

func to_base64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// Alias: to_base64
func to_base64_bytes(str []byte) string {
	return base64.StdEncoding.EncodeToString(str)
}

// Encode the string to hex
// Alias: (hex is already a function in SQLite)
func to_hex(str string) string {
	return hex.EncodeToString([]byte(str))
}

// Alias: hex, to_hex
func to_hex_bytes(str []byte) string {
	return hex.EncodeToString(str)
}

// Decode the hex encoded string
// Alias: (unhex is already a function in SQLite)
func from_hex(str string) string {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// Alias: from_hex
func from_hex_bytes(str []byte) string {
	bytes, err := hex.DecodeString(string(str))
	if err != nil {
		return ""
	}
	return string(bytes)
}

// Decode the string with the specified encoding
func decode(str string, encoding string) string {
	switch encoding {
	case "base64":
		return from_base64(str)
	case "hex":
		return from_hex(str)
	default:
		return ""
	}
}

// Alias: decode
func decode_bytes(str []byte, encoding string) string {
	switch encoding {
	case "base64":
		return from_base64_bytes(str)
	case "hex":
		return from_hex_bytes(str)
	default:
		return ""
	}
}

// Encode the string with the specified encoding
func encode(str string, encoding string) string {
	switch encoding {
	case "base64":
		return to_base64(str)
	case "hex":
		return to_hex(str)
	default:
		return ""
	}
}

// Alias: encode
func encode_bytes(str []byte, encoding string) string {
	switch encoding {
	case "base64":
		return to_base64_bytes(str)
	case "hex":
		return to_hex_bytes(str)
	default:
		return ""
	}
}

// Insert the string into the specified position with the specified length
func insert(str string, index int, length int, newStr string) string {
	if index < 1 || index > len(str) {
		return str
	}
	if length < 0 {
		return str
	}
	if index+length > len(str) {
		length = len(str) - index
	}
	return str[:index] + newStr + str[index+length:]
}

// Returns the index of the first occurrence of the substring
// Alias: position
func locate(substr string, str string) int {
	return strings.Index(str, substr) + 1
}

// Returns the index of the first occurrence of the substring starting from the specified position
// Alias: position, locate
func locate_from(substr string, str string, index int) int {
	if index < 1 || index > len(str) {
		return 0
	}
	return strings.Index(str[index:], substr) + 1
}

// Convert the string to lower case
// Alias: lower
func lcase(str string) string {
	return strings.ToLower(str)
}

// Returns the leftmost characters of the string
func left(str string, length int) string {
	if length < 0 {
		return ""
	}
	if length > len(str) {
		length = len(str)
	}
	return str[:length]
}

// Read the file and return its content
func load_file_bytes(filename string) []byte {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	return content
}

func load_file(filename string) string {
	return string(load_file_bytes(filename))
}

// Pad the string with the specified character to the specified length
func lpad(str string, length int, padStr string) string {
	if length < 0 {
		return ""
	}
	if length < len(str) {
		return str[:length]
	}
	return strings.Repeat(padStr, length-len(str)/len(padStr)) + str
}

// Pad the string with the specified character to the specified length
func rpad(str string, length int, padStr string) string {
	if length < 0 {
		return ""
	}
	if length < len(str) {
		return str[:length]
	}
	return str + strings.Repeat(padStr, length-len(str)/len(padStr))
}

// Already implemented in SQLite
/* // Trim the specified character from the left side of the string
func ltrim(str string, trimStr string) string {
	return strings.TrimLeft(str, trimStr)
}

// Trim the whitespace from the left side of the string
// Alias: ltrim
func ltrim_ws(str string) string {
	return strings.TrimLeft(str, " \t\n\r")
}

// Trim the specified character from the right side of the string
func rtrim(str string, trimStr string) string {
	return strings.TrimRight(str, trimStr)
}

// Trim the whitespace from the right side of the string
// Alias: rtrim
func rtrim_ws(str string) string {
	return strings.TrimRight(str, " \t\n\r")
} */

// Returns the length in octets of the string
func octet_length(str string) int {
	return len(str)
}

// Returns the length in octets of the bytes
func octet_length_bytes(str []byte) int {
	return len(str)
}

// Returns the octal representation of the number
// Alias: oct
func to_octal(num int) string {
	return strconv.FormatInt(int64(num), 8)
}

// Already implemented in SQLite
/* // Transform a string in a safe format for SQL
func quote(str string) string {
	returnValue := strings.Builder{}
	returnValue.WriteRune('\'')
	for _, c := range str {
		switch c {
		case '\'':
			returnValue.WriteString("''")
		case '\\':
			returnValue.WriteString("\\\\")
		case '\000':
			returnValue.WriteString("\\0")
		case '\b':
			returnValue.WriteString("\\b")
		case '\n':
			returnValue.WriteString("\\n")
		case '\r':
			returnValue.WriteString("\\r")
		case '\t':
			returnValue.WriteString("\\t")
		case '\032':
			returnValue.WriteString("\\Z")
		default:
			returnValue.WriteRune(c)
		}
	}
	returnValue.WriteRune('\'')

	return returnValue.String()
} */

// Replace the substring with the new string if match the RegExp pattern
func regexp_replace(str string, pattern string, newStr string) string {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	return regex.ReplaceAllString(str, newStr)
}

func regexp_substr(str string, pattern string) string {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	return regex.FindString(str)
}

// Repeat the string the specified number of times
func repeat(str string, count int) string {
	return strings.Repeat(str, count)
}

func right(str string, length int) string {
	if length < 0 {
		return ""
	}
	if length > len(str) {
		length = len(str)
	}
	return str[len(str)-length:]
}

// Reverse the string
func reverse(str string) string {
	runes := []rune(str)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Return the n * space string
func space(n int) string {
	return strings.Repeat(" ", n)
}

// Return the string at the upper case
// Alias: (upper is already a function in SQLite)
func ucase(str string) string {
	return strings.ToUpper(str)
}

// Slice the string from the start to the end
