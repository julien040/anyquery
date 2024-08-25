---
title: anyquery tool dev init
description: Learn how to use the anyquery tool dev init command in AnyQuery.
---

Initialize a new plugin

### Synopsis

Initialize a new plugin in the specified directory. If no directory is specified, the current directory is used.
	The module URL is the go mod URL of the plugin.

```bash
anyquery tool dev init [module URL] [dir] [flags]
```

### Examples

```bash
# Initialize a new plugin in a new directory,
anyquery tool dev init github.com/julien040/anyquery/plugins/voynich-manuscript voynich-manuscript
```

### Options

```bash
  -h, --help   help for init
```

### SEE ALSO

* [anyquery tool dev](../anyquery_tool_dev)	 - Development tools
