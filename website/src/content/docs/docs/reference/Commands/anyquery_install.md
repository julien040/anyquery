---
title: anyquery install
description: Learn how to use the anyquery install command in AnyQuery.
---

Search and install a plugin

### Synopsis

Search and install a plugin
If a plugin is specified, it will be installed without searching
If the plugin is already installed, it will fail

```bash
anyquery install [registry] [plugin] [flags]
```

### Examples

```bash
anyquery plugin install github
```

### Options

```bash
  -c, --config string   Path to the configuration database
      --csv             Output format as CSV
      --format string   Output format (pretty, json, csv, plain)
  -h, --help            help for install
      --json            Output format as JSON
      --plain           Output format as plain text
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
