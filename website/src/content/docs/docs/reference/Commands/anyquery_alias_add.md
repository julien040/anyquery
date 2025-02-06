---
title: anyquery alias add
description: Learn how to use the anyquery alias add command in Anyquery.
---

Add an alias

### Synopsis

Add an alias.
The alias name must be unique and not already used by a table.

```bash
anyquery alias add [alias] [table] [flags]
```

### Examples

```bash
anyquery alias add myalias mytable
```

### Options

```bash
  -h, --help   help for add
```

### Options inherited from parent commands

```bash
  -c, --config string   Path to the configuration database
```

### SEE ALSO

* [anyquery alias](../anyquery_alias)	 - Manage the aliases
