package namespace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/julien040/anyquery/module"
)

// TestMySQLServerSandbox exercises the real attack surface: the MySQL protocol
// handler with MustCatchMySQLSpecific, under a sandbox policy. It asserts both
// directions — the PoCs are blocked, and legitimate server functionality
// (notably the lazy in-memory information_schema ATTACH, which runs on a
// connection that already has the authorizer installed) still works.
func TestMySQLServerSandbox(t *testing.T) {
	allowed := t.TempDir()
	csvPath := filepath.Join(allowed, "ok.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte("name,age\nalice,30\n"), 0o644))

	tmp := t.TempDir()
	attackDB := filepath.Join(tmp, "pwn.db")
	vacDB := filepath.Join(tmp, "vac.db")

	ns, err := NewNamespace(NamespaceConfig{
		InMemory:     true,
		Restrictions: &module.Restrictions{AllowedDirs: []string{allowed}},
	})
	require.NoError(t, err, "creating a sandboxed namespace should not fail")

	db, err := ns.Register("sbtestdb")
	require.NoError(t, err, "registering should not fail")

	logger := log.Default()
	logger.SetOutput(io.Discard)
	if testing.Verbose() {
		logger = log.New(os.Stderr)
	}

	const addr = "127.0.0.1:8011"
	server := MySQLServer{
		DB:                     db,
		MustCatchMySQLSpecific: true,
		Address:                addr,
		Logger:                 logger,
	}
	go func() {
		_ = server.Start()
		db.Close()
	}()
	defer server.Stop()
	time.Sleep(200 * time.Millisecond)

	conn, err := sqlx.Open("mysql", "testuser:aa@tcp("+addr+")/sbtestdb")
	require.NoError(t, err, "connecting should not fail")
	defer conn.Close()

	// --- Direction 1: the PoCs are blocked over the MySQL protocol ---

	t.Run("LFR denied", func(t *testing.T) {
		_, err := conn.Exec("CREATE VIRTUAL TABLE passwd USING csv_reader('/etc/passwd')")
		require.Error(t, err, "reading /etc/passwd must be denied")
		require.Contains(t, err.Error(), "sandbox")
	})

	t.Run("ATTACH denied", func(t *testing.T) {
		_, err := conn.Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS pwn", attackDB))
		require.Error(t, err, "ATTACH to an arbitrary path must be denied")
		_, statErr := os.Stat(attackDB)
		require.True(t, os.IsNotExist(statErr), "ATTACH must not create a file")
	})

	t.Run("VACUUM INTO denied", func(t *testing.T) {
		_, err := conn.Exec(fmt.Sprintf("VACUUM main INTO '%s'", vacDB))
		require.Error(t, err, "VACUUM INTO an arbitrary path must be denied")
		_, statErr := os.Stat(vacDB)
		require.True(t, os.IsNotExist(statErr), "VACUUM INTO must not create a file")
	})

	// --- Direction 2: legitimate server functionality still works ---

	t.Run("plain query works", func(t *testing.T) {
		var n int
		require.NoError(t, conn.Get(&n, "SELECT 1 FROM dual"))
		require.Equal(t, 1, n)
	})

	t.Run("SHOW TABLES works under sandbox", func(t *testing.T) {
		// This drives the lazy in-memory information_schema/mysql ATTACH on a
		// connection that already has the sandbox authorizer installed. If the
		// in-memory allowance regressed, this would fail with "not authorized".
		var tables []string
		require.NoError(t, conn.Select(&tables, "SHOW TABLES"))
	})

	t.Run("information_schema query works under sandbox", func(t *testing.T) {
		var n int
		require.NoError(t, conn.Get(&n, "SELECT count(*) FROM information_schema.tables"))
	})

	t.Run("allowed-dir read works under sandbox", func(t *testing.T) {
		_, err := conn.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE ok USING csv_reader('%s', header=true)", csvPath))
		require.NoError(t, err, "reading a file inside an allowed dir should work")
		var n int
		require.NoError(t, conn.Get(&n, "SELECT count(*) FROM ok"))
		require.Equal(t, 1, n)
	})
}
