package registry

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
	Description string `json:"desc"`
	Author      string `json:"author"`
	Homepage    string `json:"homepage"`
	License     string `json:"license"`
	LastVersion string `json:"last_version"`
	// If type is different than anyquery or SharedObject, it will be ignored
	Type     string          `json:"type"`
	Versions []PluginVersion `json:"versions"`
}

type PluginVersion struct {
	Version                string                `json:"version"`
	MinimumRequiredVersion string                `json:"minimum_required_version"`
	Files                  map[string]PluginFile `json:"files"` // Platform -> File
	UserConfig             []UserConfig          `json:"user_config"`
	Tables                 []string              `json:"tables"`
}

type PluginFile struct {
	Hash string `json:"hash"`
	URL  string `json:"url"`
	Path string `json:"path"`
}

type UserConfig struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}
