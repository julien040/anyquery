---
title: anyquery plugins update
description: Learn how to use the anyquery plugins update command in Anyquery.
---

Update n or all plugins

### Synopsis

Update plugins
If no plugin is specified, all plugins will be updated

```bash
anyquery plugins update [...plugin] [flags]
```

### Examples

```bash
anyquery registry refresh && anyquery plugin update github
```

### Options

```bash
  -h, --help   help for update
```

### Options inherited from parent commands

```bash
  -c, --config string   Path to the configuration database
```

### SEE ALSO

* [anyquery plugins](../anyquery_plugins)	 - Print the plugins installed on the system
