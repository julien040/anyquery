---
title: anyquery profiles delete
description: Learn how to use the anyquery profiles delete command in AnyQuery.
---

Delete a profile

### Synopsis

Delete a profile.

If only two arguments are provided, we consider that the registry is the default one.
If no argument is provided, the command will prompt you the registry, the plugin and the profile to create.

```bash
anyquery profiles delete [registry] [plugin] [profile] [flags]
```

### Examples

```bash
anyquery profiles delete default github default
```

### Options

```bash
  -h, --help   help for delete
```

### Options inherited from parent commands

```bash
  -c, --config string   Path to the configuration database
```

### SEE ALSO

* [anyquery profiles](../anyquery_profiles)	 - Print the profiles installed on the system
