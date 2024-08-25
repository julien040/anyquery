---
title: anyquery alias
description: Learn how to use the anyquery alias command in AnyQuery.
---

Manage the aliases

### Synopsis

Manage the aliases.
They help you use another name for a table so that you don't have to write profileName_pluginName_tableName every time.

```bash
anyquery alias [flags]
```

### Examples

```bash
# List the aliases
anyquery alias

# Add an alias
anyquery alias add myalias mytable

# Delete an alias
anyquery alias delete myalias
```

### Options

```bash
  -c, --config string   Path to the configuration database
      --csv             Output format as CSV
      --format string   Output format (pretty, json, csv, plain)
  -h, --help            help for alias
      --json            Output format as JSON
      --plain           Output format as plain text
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
* [anyquery alias add](../anyquery_alias_add)	 - Add an alias
* [anyquery alias delete](../anyquery_alias_delete)	 - Delete an alias
* [anyquery alias list](../anyquery_alias_list)	 - List the aliases
