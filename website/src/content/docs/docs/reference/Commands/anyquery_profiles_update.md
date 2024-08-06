---
title: anyquery profiles update
description: Learn how to use the anyquery profiles update command in AnyQuery.
---

Update the profiles configuration

### Synopsis

Update the profiles configuration.

If only two arguments are provided, we consider that the registry is the default one.
If no argument is provided, the command will prompt you the registry, the plugin and the profile to update.
Note: This command requires the tty to be interactive.

```
anyquery profiles update [registry] [plugin] [profile] [flags]
```

### Options

```
  -h, --help   help for update
```

### Options inherited from parent commands

```
  -c, --config string   Path to the configuration database
```

### SEE ALSO

* [anyquery profiles](../anyquery_profiles)	 - Print the profiles installed on the system
