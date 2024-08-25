---
title: anyquery registry
description: Learn how to use the anyquery registry command in AnyQuery.
---

List the registries where plugins can be downloaded

```bash
anyquery registry [flags]
```

### Examples

```bash
anyquery registry list
```

### Options

```bash
  -c, --config string   Path to the configuration database
      --csv             Output format as CSV
      --format string   Output format (pretty, json, csv, plain)
  -h, --help            help for registry
      --json            Output format as JSON
      --plain           Output format as plain text
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
* [anyquery registry add](../anyquery_registry_add)	 - Add a new registry
* [anyquery registry get](../anyquery_registry_get)	 - Print informations about a registry
* [anyquery registry list](../anyquery_registry_list)	 - List the registries where plugins can be downloaded
* [anyquery registry refresh](../anyquery_registry_refresh)	 - Keep the registry up to date with the server
* [anyquery registry remove](../anyquery_registry_remove)	 - Remove a registry
