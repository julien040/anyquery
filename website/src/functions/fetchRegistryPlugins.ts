async function fetchRegistryPlugins(): Promise<{
    remoteQueries: { queries: query[] };
    plugins: { plugins: plugin[] };
}> {
    const fetchReg = await fetch("https://registry.anyquery.dev/v0/query/", {
        headers: {
            "Content-Type": "application/json",
        },
    });

    const remoteQueries: {
        queries: query[];
    } = await fetchReg.json();

    const pluginsFetch = await fetch(
        "https://registry.anyquery.dev/v0/registry",
        {
            headers: {
                "Content-Type": "application/json",
            },
        }
    );

    const plugins: {
        plugins: plugin[];
    } = await pluginsFetch.json();

    return { remoteQueries, plugins };
}

type query = {
    id: string;
    title: string;
    description: string;
    required_plugins: string[];
    arguments: {
        title: string;
        display_title: string;
        description: string;
        type: string;
        regex: string;
    }[];
    query: string;
    source_code: string;
    author: string;
    tags: string[];
};

type plugin = {
    name: string;
    display_name: string;
    description: string;
    icon: string;
};

export { fetchRegistryPlugins };
export type { query, plugin };
