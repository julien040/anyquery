package namespace

import (
	"database/sql"
	"io"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	_ "github.com/go-sql-driver/mysql"
)

const authFileVitessJSON = `
{
	"anyquery": [
	  {
		"UserData": "anyquery",
		"MysqlNativePassword": "*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19"
	  },
	  {
		"UserData": "anyquery",
		"Password": "thisisapassword"
	  }
	],
	"myuser": [
	  {
		"UserData": "myuser",
		"MysqlNativePassword": "*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19"
	  }
	]
  }
`

func TestMySQLAuthentication(t *testing.T) {

	t.Run("Test authentication with auth file", func(t *testing.T) {
		// Create an auth file
		file, err := os.OpenFile("auth.json", os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal("Unable to create auth file", err)
		}

		defer file.Close()
		defer os.Remove("auth.json")

		_, err = file.WriteString(authFileVitessJSON)
		if err != nil {
			t.Fatal("Unable to write to auth file", err)
		}

		namespace, err := NewNamespace(NamespaceConfig{
			InMemory: true,
		})

		if err != nil {
			t.Fatalf("Failed to create namespace: %v", err)
		}

		// Register a new database
		db, err := namespace.Register("main")

		// Create a new MySQL server
		server := MySQLServer{
			Address:                "127.0.0.1:8008",
			AuthFile:               "auth.json",
			DB:                     db,
			MustCatchMySQLSpecific: true,
		}

		if testing.Verbose() {
			server.Logger = log.Default()
		} else {
			server.Logger = log.New(io.Discard)
		}

		// Start the server
		go func() {
			server.Start()
		}()

		// Create a new MySQL client and connnect with a clear password
		time.Sleep(300 * time.Millisecond)

		errorChan := make(chan error)
		go func() {
			db, err := sql.Open("mysql", "anyquery:thisisapassword@tcp(127.0.0.1:8008)/main")
			if err != nil {
				errorChan <- err
			}
			defer db.Close()
			errorChan <- db.Ping()

		}()

		res := <-errorChan
		if res != nil {
			t.Fatalf("Failed to authenticate with auth file: %v", res)
		}

		// Create a new MySQL client and connnect with a hashed password in the auth file
		go func() {
			db, err := sql.Open("mysql", "anyquery:password@tcp(127.0.0.1:8008)/main")
			if err != nil {
				errorChan <- err
			}
			defer db.Close()
			errorChan <- db.Ping()
		}()

		res = <-errorChan
		if res != nil {
			t.Fatalf("Failed to authenticate with auth file: %v", res)
		}

		// Login with two users at the same time
		go func() {
			db, err := sql.Open("mysql", "anyquery:password@tcp(127.0.0.1:8008)/main")
			if err != nil {
				errorChan <- err
			}
			defer db.Close()
			errorChan <- db.Ping()
		}()

		go func() {
			db, err := sql.Open("mysql", "myuser:password@tcp(127.0.0.1:8008)/main")
			if err != nil {
				errorChan <- err
			}
			defer db.Close()
			errorChan <- db.Ping()
		}()

		res = <-errorChan
		if res != nil {
			t.Fatalf("Failed to authenticate with auth file: %v", res)
		}
		res = <-errorChan
		if res != nil {
			t.Fatalf("Failed to authenticate with auth file: %v", res)
		}

		t.Log("Testing wrong password and user...")

		// Test invalid password
		go func() {
			db, err := sql.Open("mysql", "anyquery:wrongpassword@tcp(127.0.0.1:8008)/main")
			if err != nil {
				errorChan <- err
			}
			defer db.Close()
			errorChan <- db.Ping()
		}()

		res = <-errorChan
		if res == nil {
			t.Fatalf("Should not have authenticated with wrong password")
		}

		// Test invalid user
		go func() {
			db, err := sql.Open("mysql", "wronguser:password@tcp(127.0.0.1:8008)/main")
			if err != nil {
				errorChan <- err
			}
			defer db.Close()
			errorChan <- db.Ping()
		}()

		res = <-errorChan
		if res == nil {
			t.Fatalf("Should not have authenticated with wrong user")
		}

		err = server.Stop()
		if err != nil {
			t.Fatalf("Failed to stop server: %v", err)
		}

	})

	t.Run("Test authentication without auth file", func(t *testing.T) {
		// Create a new MySQL server
		server := MySQLServer{
			Address: "127.0.0.1:8008",
			Logger:  log.Default(),
		}

		server.Users = make(map[string][]UserEntry)
		server.Users["myman"] = []UserEntry{
			{
				PasswordClear: "password",
			},
			{
				PasswordHash: "*9E128DA0C64A6FCCCDCFBDD0FC0A2C967C6DB36F",
			},
		}
		server.Users["myuser"] = []UserEntry{
			{
				PasswordClear: "myuserpassword",
			},
		}

		// Start the server
		go func() {
			server.Start()
		}()

		defer server.Stop()

		// Create a new MySQL client and connnect with a clear password
		time.Sleep(100 * time.Millisecond)

		// errorChan := make(chan error)

	})

}
