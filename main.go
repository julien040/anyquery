package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/julien040/anyquery/cmd"
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

	return fmt.Sprintf("%s (built with %s for %s/%s)", version, goVersion, runtime.GOOS, runtime.GOARCH)
}

func main() {

	cmd.Execute(getVersionString())

}
