package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	urlParser "net/url"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/adrg/xdg"
	"github.com/julien040/anyquery/controller/config/model"

	getter "github.com/hashicorp/go-getter"
)

// Find the highest version of a plugin that is compatible with the current version of Anyquery
// and with the user's system
func FindPluginVersionCandidate(plugin Plugin) (PluginFile, PluginVersion, error) {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	anyquerySemver, err := semver.NewVersion(anyqueryVersion)
	if err != nil {
		return PluginFile{}, PluginVersion{}, fmt.Errorf("error parsing Anyquery version: %v", err)
	}

	var candidateVersion *PluginVersion
	var candidateVersionParsed *semver.Version
	var candidateFile *PluginFile
	for _, version := range plugin.Versions {
		pluginSemver, err := semver.NewVersion(version.Version)
		if err != nil {
			continue
		}
		pluginRequiredSemver, err := semver.NewVersion(version.MinimumRequiredVersion)
		if err != nil {
			continue
		}
		// If the plugin version required is greater than the current version of Anyquery, skip
		if pluginRequiredSemver.GreaterThan(anyquerySemver) {
			continue
		}

		// Now we check if the plugin has a file for the current platform
		file, ok := version.Files[platform]
		if !ok {
			continue
		}

		// Finally, we check if this version is the highest one we found so far
		if candidateVersion == nil || pluginSemver.GreaterThan(candidateVersionParsed) {
			candidateVersion = &version
			candidateVersionParsed = pluginSemver
			candidateFile = &file
		}
	}
	if candidateVersion == nil {

		return PluginFile{}, PluginVersion{}, fmt.Errorf("no compatible version found for plugin %s", plugin.Name)
	}
	return *candidateFile, *candidateVersion, nil
}

func InstallPlugin(queries *model.Queries, registry string, plugin string) (string, error) {
	// Create the plugin directory
	path := path.Join(xdg.DataHome, "anyquery", "plugins", registry, plugin, newSmallID())
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return "", err
	}

	// Get the registry
	_, plugins, err := LoadRegistry(queries, registry)
	if err != nil {
		return "", err
	}

	// Find the plugin
	var pluginInfo *Plugin
	for _, p := range plugins.Plugins {
		if p.Name == plugin {
			pluginInfo = &p
			break
		}
	}

	// Find a compatible version
	if pluginInfo == nil {
		return "", fmt.Errorf("plugin %s not found in registry %s", plugin, registry)
	}

	file, version, err := FindPluginVersionCandidate(*pluginInfo)
	if err != nil {
		return "", err
	}
	// Download the file
	err = downloadZipToPath(file.URL, path, file.Hash)
	if err != nil {
		return "", err
	}

	// Add the plugin to the database
	var isSharedExtension int64 = 0
	if pluginInfo.Type == "sharedObject" {
		isSharedExtension = 1
	}

	configJSON, err := json.Marshal(version.UserConfig)
	if err != nil {
		return "", err
	}
	tablesJSON, err := json.Marshal(version.Tables)
	if err != nil {
		return "", err
	}
	return path, queries.AddPlugin(context.Background(), model.AddPluginParams{
		Name:     plugin,
		Registry: registry,
		Description: sql.NullString{
			String: pluginInfo.Description,
			Valid:  true,
		},
		Path:           path,
		Executablepath: file.Path,
		Version:        pluginInfo.LastVersion,
		Homepage: sql.NullString{
			String: pluginInfo.Homepage,
			Valid:  true,
		},
		Dev: 0,
		Author: sql.NullString{
			String: pluginInfo.Author,
			Valid:  true,
		},
		Issharedextension: isSharedExtension,
		Config:            string(configJSON),
		Tablename:         string(tablesJSON),
	})

}

func downloadZipToPath(url string, path string, checksum string) error {
	parsed, err := urlParser.Parse(url)
	if err != nil {
		return err
	}
	values := parsed.Query()
	values.Set("checksum", "sha256:"+checksum)
	parsed.RawQuery = values.Encode()

	client := &getter.Client{
		Src:  parsed.String(),
		Dst:  path,
		Dir:  true,
		Mode: getter.ClientModeDir,
		Ctx:  context.Background(),
	}
	return client.Get()
}

const letters = "abcdefghijklmnopqrstuvwxyz1234567890"

// Create a small ID that can be used in the path of a plugin
// It is 6 characters long and contains only letters and numbers
// It is not meant to be unique, but it is unlikely to collide
// We don't use uppercase letter due to the case-insensitive nature of some filesystems
func newSmallID() string {
	str := strings.Builder{}
	for i := 0; i < 6; i++ {
		str.WriteByte(letters[rand.IntN(len(letters))])
	}
	return str.String()
}
