package registry

import "github.com/julien040/anyquery/rpc"

/****************************************************************************************************
 *        PLEASE REFER TO THE FILE SCHEMA_PLUGIN.JSON TO KNOW THE STRUCTURE OF THE REGISTRY.        *
 *           BUT TO SUM UP, THE REGISTRY IS A JSON FILE THAT CONTAINS A LIST OF PLUGINS.            *
 * EACH PLUGIN HAS VERSIONS, AND EACH VERSION HAS FILES FOR EACH PLATFORM, USER CONFIG, AND TABLES. *
 ****************************************************************************************************/

type Registry struct {
	Title   string   `json:"title"`
	Plugins []Plugin `json:"plugins"`
}

type Plugin struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PageContent string `json:"page_content"`
	Author      string `json:"author"`
	Homepage    string `json:"homepage"`
	License     string `json:"license"`
	Registry    string // This is not a field in the registry, but can be filled for easier access
	// If type is different than anyquery or SharedObject, it will be ignored
	Type     string          `json:"type"`
	Versions []PluginVersion `json:"versions"`
}

type PluginVersion struct {
	Version                string                       `json:"version"`
	MinimumRequiredVersion string                       `json:"minimum_required_version"`
	Files                  map[string]PluginFile        `json:"files"` // Platform -> File
	UserConfig             []UserConfig                 `json:"user_config"`
	Tables                 []string                     `json:"tables"`
	TablesMetadata         map[string]rpc.TableMetadata `json:"tables_metadata"` // Table name -> Table metadata
}

type PluginFile struct {
	Hash string `json:"hash"`
	URL  string `json:"url"`
	Path string `json:"path"`
}

type UserConfig struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	// The type of the variable prompted to the user
	// Can be: string, int, float, bool, []string, []int, []float, []bool
	Type        string `json:"type"`
	Description string `json:"description"`
	Validation  string `json:"validation"`
}
