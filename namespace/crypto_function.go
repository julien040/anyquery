package namespace

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"math/rand/v2"

	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/blake2b"
)

// This file defines the hash functions that are available in SQL queries
//
// If the function has multiple alias, please specify the different names in the comment

func registerCryptoFunctions(conn *sqlite3.SQLiteConn) {
	var cryptoFunctions = []struct {
		name     string
		function any
		pure     bool
	}{
		{"md5", md5_hash, true},
		{"sha1", sha1_hash, true},
		{"sha256", sha256_hash, true},
		{"sha384", sha384_hash, true},
		{"sha512", sha512_hash, true},
		{"blake2b", blake2b_hash, true},
		{"blake2b_384", blake2b_384_hash, true},
		{"blake2b_512", blake2b_512_hash, true},
		{"random_float", random_float, false},
		{"random_real", random_float, false},
		{"random_double", random_float, false},
		{"randCanonical", random_float, false},
		{"rand", random_int, false},
		{"random_int", random_int, false},
		{"random_intn", random_intn, false},
		{"random_int64", random_int64, false},
		{"rand64", random_int64, false},
		{"random_int64n", random_int64n, false},
	}

	for _, f := range cryptoFunctions {
		conn.RegisterFunc(f.name, f.function, f.pure)
	}

}

// Alias: md5
func md5_hash(data string) string {
	hash := md5.Sum([]byte(data))
	return string(hash[:])
}

// Alias: sha1
func sha1_hash(data string) string {
	hash := sha1.Sum([]byte(data))
	return string(hash[:])
}

// Alias: sha256
func sha256_hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return string(hash[:])
}

// Alias: sha384
func sha384_hash(data string) string {
	hash := sha512.Sum384([]byte(data))
	return string(hash[:])
}

// Alias: sha512
func sha512_hash(data string) string {
	hash := sha512.Sum512([]byte(data))
	return string(hash[:])
}

func blake2b_hash(data string) string {
	hash := blake2b.Sum256([]byte(data))
	return string(hash[:])
}

func blake2b_384_hash(data string) string {
	hash := blake2b.Sum384([]byte(data))
	return string(hash[:])
}

func blake2b_512_hash(data string) string {
	hash := blake2b.Sum512([]byte(data))
	return string(hash[:])
}

/* ---------------------------- Random functions ---------------------------- */

func random_float() float64 {
	return rand.Float64()
}

func random_int() int {
	return rand.Int()
}

func random_intn(n int) int {
	return rand.IntN(n)
}

func random_int64() int64 {
	return rand.Int64()
}

func random_int64n(n int64) int64 {
	return rand.Int64N(n)
}
