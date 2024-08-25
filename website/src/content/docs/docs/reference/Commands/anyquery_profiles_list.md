---
title: anyquery profiles list
description: Learn how to use the anyquery profiles list command in AnyQuery.
---

List the profiles

### Synopsis

List the profiles.

If no argument is provided, the results will not be filtered.
If only one argument is provided, the results will be filtered by the registry.
If two arguments are provided, the results will be filtered by the registry and the plugin.

```bash
anyquery profiles list [registry] [plugin] [flags]
```

### Examples

```bash
anyquery profiles list
```

### Options

```bash
      --csv             Output format as CSV
      --format string   Output format (pretty, json, csv, plain)
  -h, --help            help for list
      --json            Output format as JSON
      --plain           Output format as plain text
```

### Options inherited from parent commands

```bash
  -c, --config string   Path to the configuration database
```

### SEE ALSO

* [anyquery profiles](../anyquery_profiles)	 - Print the profiles installed on the system
