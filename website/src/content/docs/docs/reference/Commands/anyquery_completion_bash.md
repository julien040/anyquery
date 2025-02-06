---
title: anyquery completion bash
description: Learn how to use the anyquery completion bash command in Anyquery.
---

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(anyquery completion bash)

To load completions for every new session, execute once:

#### Linux:

	anyquery completion bash > /etc/bash_completion.d/anyquery

#### macOS:

	anyquery completion bash > $(brew --prefix)/etc/bash_completion.d/anyquery

You will need to start a new shell for this setup to take effect.


```bash
anyquery completion bash
```

### Options

```bash
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [anyquery completion](../anyquery_completion)	 - Generate the autocompletion script for the specified shell
