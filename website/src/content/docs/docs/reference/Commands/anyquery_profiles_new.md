---
title: anyquery profiles new
description: Learn how to use the anyquery profiles new command in Anyquery.
---

Create a new profile

### Synopsis

Create a new profile.

If only two arguments are provided, we consider that the registry is the default one.
If no argument is provided, the command will prompt you the registry, the plugin and the profile to create.
Note: This command requires the tty to be interactive.

```bash
anyquery profiles new [registry] [plugin] [profile] [flags]
```

### Examples

```bash
anyquery profiles new default github default
```

### Options

```bash
  -h, --help   help for new
```

### Options inherited from parent commands

```bash
  -c, --config string   Path to the configuration database
```

### SEE ALSO

* [anyquery profiles](../anyquery_profiles)	 - Print the profiles installed on the system
