# SQLparser

The SQLparser extract from vitess.io project. It adds the following features:

- Tables functions support (e.g. `select * from table1(arg1, arg2)`)

## Compilation

Go to the sqlparser directory and run the following command:

```bash
go run goyacc/goyacc.go -fo sql.go sql.y
```

## Additional notes

- Tests are failing for this package because they are adapted to the vitess.io project. You can ignore them.
