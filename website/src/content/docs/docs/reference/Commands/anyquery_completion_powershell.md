---
title: anyquery completion powershell
description: Learn how to use the anyquery completion powershell command in Anyquery.
---

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	anyquery completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```bash
anyquery completion powershell [flags]
```

### Options

```bash
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [anyquery completion](../anyquery_completion)	 - Generate the autocompletion script for the specified shell
