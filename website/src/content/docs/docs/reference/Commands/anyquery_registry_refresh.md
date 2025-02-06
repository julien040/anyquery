---
title: anyquery registry refresh
description: Learn how to use the anyquery registry refresh command in Anyquery.
---

Keep the registry up to date with the server

### Synopsis

This command will fetch the registry and save the available plugins for download in the configuration database.
If a name is provided, only this registry will be refreshed. Otherwise, all registries will be refreshed.

```bash
anyquery registry refresh [name] [flags]
```

### Examples

```bash
anyquery registry refresh
```

### Options

```bash
  -h, --help   help for refresh
```

### Options inherited from parent commands

```bash
  -c, --config string   Path to the configuration database
```

### SEE ALSO

* [anyquery registry](../anyquery_registry)	 - List the registries where plugins can be downloaded
