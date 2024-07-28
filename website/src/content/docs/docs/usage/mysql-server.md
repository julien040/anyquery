---
title: MySQL Server
description: Learn how to let anyquery act as a MySQL server.
---

Anyquery can act as a MySQL server, allowing you to connect to it using any MySQL client. This is useful if the shell client is not sufficient for your needs.

## Launching the MySQL server

To launch the MySQL server, you need to use the command `server`. It will start the server on `127.0.0.1:8070` without any authentication by default.

```bash title="Launch the MySQL server"
anyquery server
```

## Configuring the MySQL server

### Changing the address and port

You can change the address and port of the MySQL server by using the `--host` and `--port` flags.

```bash title="Change the address and port of the MySQL server"
anyquery server --host "127.0.0.1" --port 3306
```

### Adding authentication

Anyquery supports the [file based authentication of Vitess](https://vitess.io/docs/20.0/user-guides/configuration-advanced/static-auth/). You can pass the path to the authentication file using the `--auth-file` flag.

```bash title="Launch the MySQL server with authentication"
anyquery server --auth-file "/path/to/auth-file"
```

The format of the authentication file is as follows:

```json
{
  "myuser": [
    {
      "MysqlNativePassword": "*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19",
      "UserData": "mydata"
    }
  ],
  {
    "myuser2": [
      {
        "Password": "mypassword2",
        "UserData": "mydata2"
      },
      {
        "Password": "mypassword3",
        "UserData": "mydata2"
      }
    ]
  }
}
```

If storing clear text passwords is not an issue, or you want to use caching_sha2_password, fill in the `Password` in clear text. Otherwise, you can use the `MysqlNativePassword` field to store the hashed password.

To generate the hashed password, the command `anyquery tool mysql-password` reads from the standard input and outputs the hashed password. Otherwise, you can launch it standalone and input the password.

```bash title="Generate hashed password from standard input"
echo "mypassword" | anyquery tool mysql-password
```

```ansi title="Generate hashed password with interactive prompt"
(base) julien@MacBook-Air-Julien anyquery % anyquery tool mysql-password   
Password: mypassword 
*FABE5482D5AADF36D028AC443D117BE1180B9725
```

### Changing the log level, file and format

By default, the server outputs logs to the standard output with the `info` level pretty printed. You can change the log level, file and format using the `--log-level`, `--log-file` and `--log-format` flags.

```bash
anyquery server --log-level "debug" --log-file "/path/to/log-file" --log-format "json"
```

**Log levels**: `debug`, `info`, `warn`, `error`, `fatal`

**Log formats**: `text`, `json`

### Changing the opened database

By default, the server opens the database `anyquery.db`. You can change the opened database using the `--database` flag. You can also attach multiple databases by using the [ATTACH](https://www.sqlite.org/lang_attach.html) statement of SQLite.

```bash title=Change the opened database
anyquery server --database "mydb"
```

You can open the database in read-only mode using the `--readonly` flag.

```bash title=Open the database in read-only mode
anyquery server --readonly
```

Finally, you can open the database in-memory using the `--in-memory` flag.

```bash title=Open the database in-memory
anyquery server --in-memory
```
