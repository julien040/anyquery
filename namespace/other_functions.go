package namespace

import (
	"os"
	pathlib "path"

	"github.com/adrg/xdg"
	"github.com/julien040/anyquery/module"
	"github.com/mattn/go-sqlite3"
)

/* ------------------------------- Clear cache ------------------------------ */

func registerOtherFunctions(conn *sqlite3.SQLiteConn) {
	var otherFunctions = []struct {
		name     string
		function any
		pure     bool
	}{
		{"clear_file_cache", clear_file_cache, true},
		{"clear_plugin_cache", clear_plugin_cache, true},
	}
	for _, f := range otherFunctions {
		conn.RegisterFunc(f.name, f.function, f.pure)
	}
}

func clear_file_cache() string {
	pathToRemove := pathlib.Join(xdg.CacheHome, "anyquery", "downloads")

	// Remove the directory
	err := os.RemoveAll(pathToRemove)
	if err != nil {
		return err.Error()
	}

	return ""
}

func clear_plugin_cache(plugin string) string {
	pathToRemove := pathlib.Join(xdg.CacheHome, "anyquery", "plugins", plugin)

	if plugin == "" {
		return "The plugin name is empty"
	}

	// Remove the directory
	err := os.RemoveAll(pathToRemove)
	if err != nil {
		return err.Error()
	}

	return ""
}

type bufferFlusher struct {
	modules *map[string]*module.SQLiteModule
}

func (b *bufferFlusher) Clear(plugin string) string {
	if plugin == "" {
		return "The plugin name is empty"
	}

	if mod, ok := (*b.modules)[plugin]; !ok {
		return "The plugin does not exist"
	} else {
		if mod.Table == nil {
			return "The plugin is not loaded"
		}

		mod.Table.ClearBuffers()
	}

	return ""
}

func (b *bufferFlusher) Flush(plugin string) string {
	if plugin == "" {
		return "The plugin name is empty"
	}

	if mod, ok := (*b.modules)[plugin]; !ok {
		return "The plugin does not exist"
	} else {
		if mod.Table == nil {
			return "The plugin is not loaded"
		}

		err := mod.Table.FlushBuffers()
		if err != nil {
			return err.Error()
		}
	}

	return ""
}
