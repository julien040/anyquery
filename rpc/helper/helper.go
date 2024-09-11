package helper

import (
	"path"

	"github.com/adrg/xdg"
)

// Find the directory where the plugin can cache data
//
// It is recommended to use this function to get the cache directory
// because the SQL function clear_plugin_cache(plugin_name) will clear
// the directory returned by this function
//
// Internally, this function uses the XDG_CACHE_HOME environment variable
// so that anyone can override the cache directory to its needs
func GetCachePath(pluginName string) string {
	return path.Join(xdg.CacheHome, "anyquery", "plugins", pluginName)
}
