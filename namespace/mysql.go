package namespace

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/charmbracelet/log"

	"vitess.io/vitess/go/mysql"
)

type UserEntry struct {
	// The clear password of the user used to authenticate
	PasswordClear string

	// The native hashed password of the user used to authenticate
	// (recommended to be used instead of PasswordClear)
	//
	// It is the HEX(sha1(sha1(password))) of the password prefixed by "*"
	PasswordHash string
}

// Represent a MySQL-compatible server to run queries on a sql.DB instance
type MySQLServer struct {
	// The address of the server to bind to
	// (e.g. "localhost:3306")
	Address string

	// Auth file is a path to a file that contains
	// the username and passwords to use for the server
	//
	// It follows the format of the Vitess' file based authentication
	// which can be found at:
	// https://vitess.io/docs/19.0/user-guides/configuration-advanced/static-auth/
	//
	// Note: for safety reasons, the file should be readable only by the user
	// running the server
	AuthFile string

	// A map of users that can be used to authenticate to the server
	//
	// The key is the username and the value is an array of UserEntry.
	// Therefore, a user can have multiple passwords to authenticate
	//
	// If AuthFile is provided, this field will be ignored
	// If neither AuthFile nor Users are provided, the server will accept any connection
	Users map[string][]UserEntry

	// SQLite doesn't use the same exact dialect as MySQL.
	// For example, queries like "SHOW TABLES", "SET", "SHOW DATABASES", "USE"
	// are not supported by SQLite and will return an error.
	//
	// If this field is set to true, the server will catch these MySQL-specific
	// queries and return an adequate answer so that MySQL clients can work with it.
	//
	// Note: this is a best-effort implementation and some queries might not work as expected
	MustCatchMySQLSpecific bool

	// The struct from vitess that will be used to listen for incoming connections
	listener *mysql.Listener

	// If the server has been started
	//
	// This is used to prevent the server from being started multiple times
	serverStarted bool

	// The handler that will be passed to the listener
	handler handler

	// The database connection to SQLite used by the server
	//
	// When the server is closed, the connection will not be closed
	// and it is the responsibility of the caller to close it
	DB *sql.DB

	// The logger used by the server
	Logger *log.Logger
}

func convertUserEntriesToVitessAuthFile(users map[string][]UserEntry) (string, error) {
	var config map[string][]mysql.AuthServerStaticEntry = make(map[string][]mysql.AuthServerStaticEntry)

	// We create a map similar to the one internally used by Vitess
	// https://github.com/vitessio/vitess/blob/main/go/mysql/auth_server_static.go#L299
	for username, entries := range users {
		var userEntries []mysql.AuthServerStaticEntry
		for _, entry := range entries {
			var userEntry mysql.AuthServerStaticEntry
			userEntry.Password = entry.PasswordClear
			userEntry.MysqlNativePassword = entry.PasswordHash
			userEntry.UserData = username

			userEntries = append(userEntries, userEntry)
		}

		config[username] = userEntries
	}

	// We convert the map to a json string
	authFile, err := json.Marshal(config)
	return string(authFile), err

}

// Start the MySQL server
func (s *MySQLServer) Start() error {
	// If the server has already been started, return an error
	if s.serverStarted {
		return fmt.Errorf("server already started")
	}

	// If the address is empty, return an error
	if s.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Represent a method to authenticate users
	// against the server
	var authServer mysql.AuthServer

	if s.AuthFile == "" && s.Users == nil {
		// If no method is specified, the server accepts any connection
		// mysql package has a built-in method to accept any connection
		authServer = mysql.NewAuthServerNone()
	} else if s.AuthFile != "" {
		// If an auth file is provided, we transfer it to the auth server
		// that will read the JSON file and load the users
		// We set 0 to the reloadInterval so that the server doesn't reload the file
		authServer = mysql.NewAuthServerStatic(s.AuthFile, "", 0)
	} else if s.Users != nil {
		// If users are provided, we convert them to an auth file
		// because the auth server only accepts a JSON string
		authJSON, err := convertUserEntriesToVitessAuthFile(s.Users)
		if err != nil {
			return err
		}

		authServer = mysql.NewAuthServerStatic("", authJSON, 0)
	}

	// We create a new handler with the database connection
	s.handler = handler{
		DB:                  s.DB,
		RewriteMySQLQueries: s.MustCatchMySQLSpecific,
		Logger:              s.Logger,
	}

	// We create a new listener with the auth server
	// I have set default values that I'm not sure to properly understand
	// Feel free to open a pull request for more sensible values
	listener, err := mysql.NewListener("tcp", s.Address, authServer, &s.handler,
		0, 0, false, true, 1*time.Hour, 60*time.Second)

	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}

	s.listener = listener

	s.serverStarted = true

	// Start the listener
	s.listener.Accept()

	return nil
}

func (s *MySQLServer) Stop() error {
	if !s.serverStarted {
		return fmt.Errorf("server not started")
	}

	s.listener.Shutdown()

	// Iterate over the connections and close them
	// This is necessary because the listener doesn't close the connections

	for _, conn := range s.handler.connections {
		if !conn.IsClosed() {
			conn.Close()
		}
	}

	return nil
}
