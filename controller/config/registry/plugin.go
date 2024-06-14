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
	"github.com/julien040/go-ternary"

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
		Version:        version.Version,
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
		Checksumdir:       sql.NullString{},
	})
}

// Return a list of all the plugins installable from the registries
// It does not check if the plugin is already installed,
// neither if the plugin is compatible with the current version of Anyquery
// nor if the plugin has a file for the current platform
func ListInstallablePlugins(queries *model.Queries) ([]Plugin, error) {
	plugins := []Plugin{}
	// We get the plugins
	ctx := context.Background()
	registry, err := queries.GetRegistries(ctx)
	if err != nil {
		return plugins, fmt.Errorf("could not get the registries: %w", err)
	}

	for _, r := range registry {
		_, p, err := LoadRegistry(queries, r.Name)
		if err != nil {
			return plugins, fmt.Errorf("could not load registry %s: %w", r.Name, err)
		}

		plugins = append(plugins, p.Plugins...)
	}
	return plugins, nil
}

func GetCurrentPlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// Return a list of all the plugins installable from the registries
// for the current platform. List also the plugins that are already installed
func ListInstallablePluginsForPlatform(queries *model.Queries, platform string) ([]Plugin, error) {
	plugins, err := ListInstallablePlugins(queries)
	if err != nil {
		return nil, err
	}
	platformPlugins := []Plugin{}
	for _, plugin := range plugins {
		for _, version := range plugin.Versions {
			// Check if the version is compatible with the current version of Anyquery
			pluginVersion, err := semver.NewVersion(version.MinimumRequiredVersion)
			if err != nil {
				continue
			}

			if pluginVersion.GreaterThan(anyqueryParsedVersion) {
				continue
			}

			// Check if the plugin has a file for the platform
			// and the version is compatible with the current version of Anyquery
			_, ok := version.Files[platform]
			if ok {
				platformPlugins = append(platformPlugins, plugin)
				break
			}

		}
	}
	return platformPlugins, nil
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
// It is not meant to be unique, but it is unlikely to have a collision (308 915 776 possibilities)
// We don't use uppercase letter due to the case-insensitive nature of some filesystems
func newSmallID() string {
	str := strings.Builder{}
	for i := 0; i < 6; i++ {
		str.WriteByte(letters[rand.IntN(len(letters))])
	}
	return str.String()
}

func UninstallPlugin(queries *model.Queries, registry string, plugin string) error {
	// Get the plugin
	pluginInfo, err := queries.GetPlugin(context.Background(), model.GetPluginParams{
		Name:     plugin,
		Registry: registry,
	})
	if err != nil {
		return fmt.Errorf("could not get the plugin: %w", err)
	}

	// Ensure the plugin is not linked to any profile
	linkedProfiles, err := queries.GetProfilesOfPlugin(context.Background(), model.GetProfilesOfPluginParams{
		Pluginname: plugin,
		Registry:   registry,
	})
	if err != nil {
		return fmt.Errorf("could not get the linked profiles: %w", err)
	}
	if len(linkedProfiles) > 0 {
		return fmt.Errorf("the plugin is linked to profiles: %v", linkedProfiles)
	}

	// Uninstall the plugin
	err = os.RemoveAll(pluginInfo.Path)
	if err != nil {
		return fmt.Errorf("could not remove the plugin: %w", err)
	}

	// Remove the plugin from the database
	err = queries.DeletePlugin(context.Background(), model.DeletePluginParams{
		Name:     plugin,
		Registry: registry,
	})
	if err != nil {
		return fmt.Errorf("could not remove the plugin from the database: %w", err)
	}
	return nil
}

func UpdatePlugin(queries *model.Queries, registry string, plugin string) error {
	// Get the plugin
	pluginInfo, err := queries.GetPlugin(context.Background(), model.GetPluginParams{
		Name:     plugin,
		Registry: registry,
	})
	if err != nil {
		return fmt.Errorf("could not get the plugin: %w", err)
	}

	// Find the plugin in the registry
	_, plugins, err := LoadRegistry(queries, registry)
	if err != nil {
		return err
	}
	var pluginInfoRegistry *Plugin
	for _, p := range plugins.Plugins {
		if p.Name == plugin {
			pluginInfoRegistry = &p
			break
		}
	}

	if pluginInfoRegistry == nil {
		return fmt.Errorf("plugin %s not found in registry %s", plugin, registry)
	}

	// Find a compatible version
	file, version, err := FindPluginVersionCandidate(*pluginInfoRegistry)
	if err != nil {
		return err
	}
	// Download the file
	err = downloadZipToPath(file.URL, pluginInfo.Path, file.Hash)
	if err != nil {
		return err
	}

	// Update the plugin in the database
	configJSON, err := json.Marshal(version.UserConfig)
	if err != nil {
		return err
	}
	tablesJSON, err := json.Marshal(version.Tables)
	if err != nil {
		return err
	}
	err = queries.UpdatePlugin(context.Background(), model.UpdatePluginParams{
		Name:     plugin,
		Registry: registry,
		Description: sql.NullString{
			String: pluginInfoRegistry.Description,
			Valid:  true,
		},
		Version: version.Version,
		Homepage: sql.NullString{
			String: pluginInfoRegistry.Homepage,
			Valid:  true,
		},
		Executablepath: file.Path,
		Config:         string(configJSON),
		Checksumdir:    sql.NullString{},
		Tablename:      string(tablesJSON),
		Author: sql.NullString{
			String: pluginInfoRegistry.Author,
			Valid:  true,
		},
		Issharedextension: int64(ternary.If(pluginInfoRegistry.Type == "sharedObject", 1, 0)),
	})
	return err
}
