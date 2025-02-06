---
title: anyquery completion fish
description: Learn how to use the anyquery completion fish command in Anyquery.
---

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	anyquery completion fish | source

To load completions for every new session, execute once:

	anyquery completion fish > ~/.config/fish/completions/anyquery.fish

You will need to start a new shell for this setup to take effect.


```bash
anyquery completion fish [flags]
```

### Options

```bash
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [anyquery completion](../anyquery_completion)	 - Generate the autocompletion script for the specified shell
