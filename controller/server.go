package controller

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/namespace"
	"github.com/spf13/cobra"
)

func Server(cmd *cobra.Command, args []string) error {

	// Get the flags
	var anyqueryConfigPath, host, logLevel, logFormat, logFile, authfile, path string
	var err error
	var readOnly, inMemory bool
	var port int

	host, _ = cmd.Flags().GetString("host")
	if host == "" {
		// If no host is provided, we default to the loopback interface
		// localhost seems to have issues on some systems
		host = "127.0.0.1"
	}

	port, _ = cmd.Flags().GetInt("port")
	if port == 0 {
		port = 8070
	}

	readOnly, _ = cmd.Flags().GetBool("readonly")
	inMemory, _ = cmd.Flags().GetBool("in-memory")
	path, _ = cmd.Flags().GetString("database")

	anyqueryConfigPath, err = cmd.Flags().GetString("config")
	if anyqueryConfigPath == "" {
		anyqueryConfigPath, err = xdg.ConfigFile("anyquery/config.db")
		if err != nil {
			return err
		}
	}

	logLevel, _ = cmd.Flags().GetString("log-level")

	// Create the logger and set the log level
	lo := log.Default()
	switch logLevel {
	case "debug":
		lo.SetLevel(log.DebugLevel)
	case "info":
		lo.SetLevel(log.InfoLevel)
	case "warn":
		lo.SetLevel(log.WarnLevel)
	case "error":
		lo.SetLevel(log.ErrorLevel)
	case "fatal":
		lo.SetLevel(log.FatalLevel)
	default:
		lo.SetLevel(log.InfoLevel)
	}

	logFormat, _ = cmd.Flags().GetString("log-format")

	switch logFormat {
	case "json":
		lo.SetFormatter(log.JSONFormatter)
	default:
		lo.SetFormatter(log.TextFormatter)
	}

	logFile, _ = cmd.Flags().GetString("log-file")
	switch logFile {
	case "/dev/stdout":
		lo.SetOutput(os.Stdout)
	case "/dev/stderr":
		lo.SetOutput(os.Stderr)
	default:
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("could not open log file: %w", err)
		}
		lo.SetOutput(file)
	}

	authfile, _ = cmd.Flags().GetString("auth-file")

	// Set the logger for the plugin
	loPlugin := lo.WithPrefix("plugin")

	// Create the namespace
	instance, err := namespace.NewNamespace(namespace.NamespaceConfig{
		InMemory: inMemory,
		ReadOnly: readOnly,
		Path:     path,
		Logger: hclog.FromStandardLogger(loPlugin.StandardLog(), &hclog.LoggerOptions{
			Level:       hclog.LevelFromString(logLevel),
			DisableTime: true,
		}),
	})
	if err != nil {
		return err
	}

	err = instance.LoadAsAnyqueryCLI(anyqueryConfigPath)
	if err != nil {
		return err
	}

	// We register the namespace
	db, err := instance.Register("")
	if err != nil {
		lo.Fatal("could not register namespace", "error", err)
	}
	// defer db.Close()

	// We create the server
	mySQLServer := namespace.MySQLServer{
		Logger:                 lo,
		DB:                     db,
		MustCatchMySQLSpecific: true,
		Address:                fmt.Sprintf("%s:%d", host, port),
		AuthFile:               authfile,
	}

	dsn := ""
	if authfile != "" {
		dsn = fmt.Sprintf("username:password@tcp(%s)/main", mySQLServer.Address)
	} else {
		dsn = fmt.Sprintf("tcp(%s)/main", mySQLServer.Address)
	}

	lo.Info("Starting server", "address", mySQLServer.Address, "connectionString", dsn)

	// We catch the signals to stop the server
	// to do a clean shutdown
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-osSignal
		mySQLServer.Stop()
	}()

	// We start the server
	err = mySQLServer.Start()
	if err != nil {
		lo.Fatal("Server stopped", "error", err)
	} else {
		lo.Info("Server stopped")
	}

	lo.Info("Exiting in 10 seconds")

	go func() {
		err = db.Close()
		if err != nil {
			lo.Error("could not close database", "error", err)
		}
	}()

	// We wait 10 seconds before exiting
	// to give the server time to close
	time.Sleep(10 * time.Second)

	return nil
}
