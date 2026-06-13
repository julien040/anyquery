// Types describing the response of the Anyquery registry API
// (https://registry.anyquery.dev/v0/registry/).

/** Platform identifiers a plugin can be built for. */
export type Platform =
    | "darwin/amd64"
    | "darwin/arm64"
    | "linux/amd64"
    | "linux/arm64"
    | "windows/amd64"
    | "windows/arm64";

/** Types a user configuration field can take. */
export type UserConfigType =
    | "string"
    | "int"
    | "float"
    | "boolean"
    | "[]string"
    | "[]int"
    | "[]float"
    | "[]boolean";

/** A downloadable build of a plugin version for a given platform. */
export type PluginFile = {
    hash: string;
    url: string;
    path: string;
};

/** A single field the user must/can configure when installing a plugin. */
export type UserConfig = {
    name: string;
    required: boolean;
    type: UserConfigType;
    description: string;
};

/** Documentation about a single table exposed by a plugin version. */
export type TableMetadata = {
    description: string;
    examples: string[];
};

/** A released version of a plugin. */
export type PluginVersion = {
    version: string;
    files: Partial<Record<Platform, PluginFile>>;
    minimum_required_version: string;
    user_config: UserConfig[];
    tables: string[];
    tables_metadata: Record<string, TableMetadata>;
};

/** A plugin entry in the registry. */
export type Plugin = {
    name: string;
    display_name: string;
    description: string;
    author: string;
    versions: PluginVersion[];
    license: string;
    homepage: string;
    last_version: string;
    type: string;
    icon: string;
    page_content: string;
};

/** The full response returned by the registry endpoint. */
export type RegistryResponse = {
    plugins: Plugin[];
};
