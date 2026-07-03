package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/julien040/anyquery/cmd"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
)

// Version of the program
// Can be replaced by the build system
var version = "dev"

// Get the current package version and the go version
// If the program builder didn't replace main.version, the function will try to
// replace it with the package version
func getVersionString() string {
	goVersion := runtime.Version()
	buildDebug, ok := debug.ReadBuildInfo()
	// Replace only if from go install and not an official release
	if ok && buildDebug.Main.Version != "(devel)" && buildDebug.Main.Version != "" && version == "dev" {
		version = buildDebug.Main.Version
	}

	// Get the SQLite version
	sqliteVersion, _, _ := sqlite3.Version()

	return fmt.Sprintf("%s (built with %s for %s/%s) · SQLite %s", version, goVersion, runtime.GOOS, runtime.GOARCH, sqliteVersion)
}

func main() {

	// getVersionString resolves the package `version` variable as a side effect,
	// so call it first, then pass the raw version separately (Go does not order
	// the read of `version` against the call, so we must not rely on arg order).
	display := getVersionString()
	cmd.Execute(display, version)

}
