# Upload SQLean extension to Anyquery

> ⚠️ You need to have `bun` installed to run the script.

To upload the SQLean extension to Anyquery, you need to update the variable `currentVersion` in the file `index.ts` with the version you want to upload.

Then, run the following commands:

```bash
bun install
chmod u+x index.ts
./index.ts
```

The script will input the user/password for the registry and upload the extension to Anyquery.

> **Note:** If a new extension is added, you need to update the `libs` array in the `index.ts` file with the new extension name and its description.

## Development notes

SQLean extension uses a loader with the symbol `_sqlite3_<plugin name>_init`. SQLite automatically replaces the `<plugin name>` with the name of the extension without the file extension. Therefore, the extension name (e.g. `uuid.so`) must match the name of the loader (e.g. `_sqlite3_uuid_init`) and not be renamed to `lib.so` or similar.
