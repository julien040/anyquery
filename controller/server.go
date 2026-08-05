package controller

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/module"
	"github.com/julien040/anyquery/namespace"
	"github.com/spf13/cobra"
)

// serverRestrictions is the sandboxing policy for `anyquery server`.
//
// --dev implies --no-sandbox and wins over every explicit sandbox flag
// (--no-sandbox, --allow-dirs, --allow-remote, --allow-attach,
// --allow-db-connections): the dev-mode UDFs (load_dev_plugin and friends,
// namespace/developer.go) read an arbitrary manifest path, exec its
// build_command, and create/append its log_file with no Restrictions check.
// Requiring operators to also pass --no-sandbox alongside --dev was rejected
// as bad CLI ergonomics ("that's dumb to write two flags"), so the
// disablement is unconditional and total rather than scoped to just the dev
// UDFs — under --dev this server also re-enables load_file, the read_*
// modules' local/remote access, on-disk ATTACH, and the database reader
// modules. See the sandbox docs and create-plugin.md for the operator-facing
// warning.
func serverRestrictions(cmd *cobra.Command) *module.Restrictions {
	if dev, _ := cmd.Flags().GetBool("dev"); dev {
		return nil
	}
	return RestrictionsFromFlags(cmd)
}

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

	// Check if the dev mode is enabled
	dev, _ := cmd.Flags().GetBool("dev")

	// Create the namespace
	restrictions := serverRestrictions(cmd)
	switch {
	case restrictions != nil:
		lo.Info("Server sandboxing enabled",
			"allowedDirs", restrictions.AllowedDirs,
			"allowRemote", restrictions.AllowRemote,
			"allowAttach", restrictions.AllowAttach,
			"allowDBConnections", restrictions.AllowDBConnections)
	case dev:
		// --dev implies --no-sandbox (serverRestrictions above), so this branch
		// is reached whenever --dev is passed, regardless of any --no-sandbox /
		// --allow-* flags also on the command line.
		lo.Warn("Server sandboxing is DISABLED (--dev): developer mode always disables the sandbox, because load_dev_plugin/reload_dev_plugin/unload_dev_plugin read arbitrary files, exec build_command, and write log_file with no policy check. Clients can also read local files, reach internal endpoints, and write arbitrary files. Do not expose this server to a network; keep --host on loopback (the default) and do not run --dev in production.")
		if isNetworkExposedHost(host) {
			lo.Warn("--dev server is bound to a non-loopback host: this is UNSAFE and unauthenticated by default", "host", host)
		}
	default:
		lo.Warn("Server sandboxing is DISABLED (--no-sandbox): clients can read local files, reach internal endpoints, and write arbitrary files")
	}

	instance, err := namespace.NewNamespace(namespace.NamespaceConfig{
		InMemory: inMemory,
		ReadOnly: readOnly,
		Path:     path,
		Logger: hclog.FromStandardLogger(loPlugin.StandardLog(), &hclog.LoggerOptions{
			Level:       hclog.LevelFromString(logLevel),
			DisableTime: true,
		}),
		DevMode:      dev,
		Restrictions: restrictions,
	})
	if err != nil {
		return err
	}

	err = instance.LoadAsAnyqueryCLI(anyqueryConfigPath)
	if err != nil {
		return err
	}

	// Get the extensions
	extensions, _ := cmd.Flags().GetStringSlice("extension")
	for _, extension := range extensions {
		err = instance.LoadSharedExtension(extension, "")
		if err != nil {
			return fmt.Errorf("failed to load extension: %w", err)
		}
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
	signal.Notify(osSignal, os.Interrupt)
	if runtime.GOOS != "windows" {
		signal.Notify(osSignal, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGHUP)
	}

	go func() {
		// Print the signal received
		sig := <-osSignal
		lo.Info("Signal received", "signal", sig)
		mySQLServer.Stop()
	}()

	// We start the server
	err = mySQLServer.Start()
	if err != nil {
		lo.Fatal("Server stopped", "error", err)
	} else {
		lo.Info("Server stopped")
	}

	lo.Info("Exiting in 5 seconds")

	go func() {
		err = db.Close()
		if err != nil {
			lo.Error("could not close database", "error", err)
		}
	}()

	// We wait 10 seconds before exiting
	// to give the server time to close
	time.Sleep(5 * time.Second)

	return nil
}
