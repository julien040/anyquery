package module

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"testing"

	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
	"github.com/parquet-go/parquet-go"
)

// pipeStdin points os.Stdin at the read end of a pipe carrying data, restoring
// the real stdin when the test ends. The write runs in a goroutine because a
// payload larger than the pipe buffer (16 KiB on macOS, 64 KiB on Linux) would
// otherwise block before anything starts draining it.
func pipeStdin(t *testing.T, data []byte) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		r.Close()
	})
	go func() {
		w.Write(data)
		w.Close()
	}()
}

// TestStdinDeniedUnderSandbox: stdin is denied with a non-nil policy, allowed
// with nil, for every reader. read_parquet has no stdin
// support at all (it needs random access), so it is expected to always
// error — asserted separately below.
func TestStdinDeniedUnderSandbox(t *testing.T) {
	type reader struct {
		name       string
		moduleName string
		newModule  func(r *Restrictions) sqlite3.Module
	}
	readers := []reader{
		{"csv", "csv_reader", func(r *Restrictions) sqlite3.Module { return &CsvModule{Restrictions: r} }},
		{"json", "json_reader", func(r *Restrictions) sqlite3.Module { return &JSONModule{Restrictions: r} }},
		{"jsonl", "jsonl_reader", func(r *Restrictions) sqlite3.Module { return &JSONlModule{Restrictions: r} }},
		{"log", "log_reader", func(r *Restrictions) sqlite3.Module { return &LogModule{Restrictions: r} }},
		{"toml", "toml_reader", func(r *Restrictions) sqlite3.Module { return &TomlModule{Restrictions: r} }},
		{"yaml", "yaml_reader", func(r *Restrictions) sqlite3.Module { return &YamlModule{Restrictions: r} }},
		{"html", "html_reader", func(r *Restrictions) sqlite3.Module { return &HtmlModule{Restrictions: r} }},
	}

	for _, rd := range readers {
		t.Run(rd.name+"/denied under sandbox", func(t *testing.T) {
			name := fmt.Sprintf("sqlite3-stdin-deny-%s", rd.name)
			sql.Register(name, &sqlite3.SQLiteDriver{
				ConnectHook: func(conn *sqlite3.SQLiteConn) error {
					return conn.CreateModule(rd.moduleName, rd.newModule(&Restrictions{}))
				},
			})
			db, err := sql.Open(name, ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			_, err = db.Exec(fmt.Sprintf("create virtual table t using %s(file='stdin')", rd.moduleName))
			if err == nil {
				t.Fatalf("%s: reading from stdin was permitted under a sandbox", rd.name)
			}
		})
	}
}

// TestStdinAllowedWithoutSandbox is the flip side: a nil policy must still
// permit stdin, so the AllowStdin guard in every reader doesn't accidentally
// deny the unrestricted case.
func TestStdinAllowedWithoutSandbox(t *testing.T) {
	pipeStdin(t, []byte("a,b\n1,2\n"))

	name := "sqlite3-stdin-allow-csv"
	sql.Register(name, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.CreateModule("csv_reader", &CsvModule{Restrictions: nil})
		},
	})
	db, err := sql.Open(name, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec("create virtual table t using csv_reader(file='stdin', header=true)")
	if err != nil {
		t.Fatalf("stdin denied without a sandbox: %v", err)
	}
	var count int
	if err := db.QueryRow("select count(*) from t").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("got %d rows, want 1", count)
	}
}

// parquetStdinFixture builds a two-row parquet file in memory. read_parquet
// needs random access, so this is the payload that Fetcher.OpenMmap has to
// spool to a file before it can be mapped.
func parquetStdinFixture(t *testing.T) []byte {
	t.Helper()
	type row struct {
		ID   int64  `parquet:"id"`
		Name string `parquet:"name"`
	}
	var buf bytes.Buffer
	w := parquet.NewGenericWriter[row](&buf)
	if _, err := w.Write([]row{{ID: 1, Name: "alpha"}, {ID: 2, Name: "beta"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func registerParquetStdinDB(t *testing.T, driverName string, r *Restrictions) *sql.DB {
	t.Helper()
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.CreateModule("parquet_reader", &ParquetModule{Restrictions: r})
		},
	})
	db, err := sql.Open(driverName, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestParquetStdin: read_parquet reads through a memory mapping, which stdin
// cannot provide directly — Fetcher.OpenMmap spools it to a file first, so
// piping a parquet file in works. The sandbox gate is unchanged: a non-nil
// policy still refuses stdin.
func TestParquetStdin(t *testing.T) {
	t.Run("denied under sandbox", func(t *testing.T) {
		pipeStdin(t, parquetStdinFixture(t))
		db := registerParquetStdinDB(t, "sqlite3-parquet-stdin-deny", &Restrictions{})
		if _, err := db.Exec("create virtual table t using parquet_reader(file='stdin')"); err == nil {
			t.Fatalf("parquet_reader('stdin') was permitted under a sandbox")
		}
	})

	t.Run("allowed without sandbox", func(t *testing.T) {
		pipeStdin(t, parquetStdinFixture(t))
		db := registerParquetStdinDB(t, "sqlite3-parquet-stdin-allow", nil)
		if _, err := db.Exec("create virtual table t using parquet_reader(file='stdin')"); err != nil {
			t.Fatalf("parquet_reader('stdin') denied without a sandbox: %v", err)
		}
		var count int
		if err := db.QueryRow("select count(*) from t").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("got %d rows, want 2", count)
		}
		var name string
		if err := db.QueryRow("select name from t where id = 2").Scan(&name); err != nil {
			t.Fatal(err)
		}
		if name != "beta" {
			t.Fatalf("got %q, want %q", name, "beta")
		}
	})
}
