# Anyquery

<img src="https://anyquery.dev/images/logo-shadow.png" alt="Anyquery logo" width="96"></img>

![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/julien040/anyquery/total)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/julien040/anyquery)
[![Documentation](https://img.shields.io/badge/documentation-blue)](https://anyquery.dev)
[![GitHub issues](https://img.shields.io/github/issues/julien040/anyquery)](https://github.com/julien040/anyquery/issues)
[![Dynamic JSON Badge](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fregistry.anyquery.dev%2Fv0%2Fregistry%2F&query=%24.plugins_count&label=Integrations%20count&cacheSeconds=3600)](https://anyquery.dev/integrations/)

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything. It supports querying [JSON](https://anyquery.dev/docs/usage/querying-files/#json), [CSV](https://anyquery.dev/docs/usage/querying-files/#csv), [Parquet](https://anyquery.dev/docs/usage/querying-files/#parquet), SQLite, [Airtable bases](https://anyquery.dev/integrations/airtable/), [Google Sheets](https://anyquery.dev/integrations/google_sheets/), [Notion databases](https://anyquery.dev/integrations/notion/), [logs file](https://anyquery.dev/docs/usage/querying-log/) using [Grok](https://www.elastic.co/guide/en/elasticsearch/reference/current/grok.html), and more. It also supports running SQL queries on [remote files](https://anyquery.dev/docs/usage/querying-files/#remote-files) (HTTP, S3, GCS) and local apps ([Apple Notes](https://anyquery.dev/integrations/notes/), [Apple Reminders](https://anyquery.dev/integrations/reminders/), [Google Chrome Tabs](https://anyquery.dev/integrations/chrome/), etc.).
It's built on top of [SQLite](https://www.sqlite.org) and uses [plugins](https://anyquery.dev/integrations/) to extend its functionality.

Moreover, it can acts as a [MySQL server](https://anyquery.dev/docs/usage/mysql-server/), allowing you to run SQL queries from your favorite MySQL-compatible client (e.g. [Looker Studio](https://anyquery.dev/connection-guide/looker-studio/), [DBeaver](https://anyquery.dev/connection-guide/dbeaver/), [TablePlus](https://anyquery.dev/connection-guide/tableplus/), [Metabase](https://anyquery.dev/connection-guide/metabase/), etc.).

![Anyquery header](https://anyquery.dev/images/release-header.png)

## Usage

The [documentation](https://anyquery.dev/docs/usage/running-queries) provides detailed instructions on how to run queries with Anyquery.
But let's see a quick example. Type `anyquery` in your terminal to open the shell mode. Then, run the following query:

```sql
-- List all repositories of asg017 related to SQLite
SELECT full_name, stargazers_count, pushed_at FROM github_repositories_from_user('asg017') WHERE name LIKE '%sqlite%';

-- Count rows of a remote 75MB CSV file
SELECT count(*) FROM read_csv('https://raw.githubusercontent.com/datadesk/california-coronavirus-data/master/latimes-place-totals.csv', header=true);

-- Insert into a Notion database all repositories of nalgeon related to SQLite
INSERT INTO notion_database(repo, stars, last_push) SELECT full_name, stargazers_count, pushed_at FROM github_repositories_from_user('nalgeon') WHERE description LIKE '%sqlite%';

-- Close all tabs of the datasette documentation
DELETE FROM chrome_tabs WHERE url LIKE '%datasette%';
```

You can also launch the MySQL server with `anyquery server` and connect to it with your favorite MySQL-compatible client.

```bash
anyquery server &
mysql -u root -h 127.0.0.1 -P 8070
```

## Installation

The [documentation](https://anyquery.dev/docs/#installation) provides detailed instructions on how to install Anyquery on your system. You can install anyquery from Homebrew, APT, YUM/DNF, Scoop, Winget and Chocolatey. You can also download the binary from the [releases page](https://github.com/julien040/anyquery/releases).

### Homebrew

```zsh
brew install julien040/anyquery/anyquery
```

### Snap

```bash
sudo snap install anyquery
```

### APT

```bash
echo "deb [trusted=yes] https://apt.julienc.me/ /" | sudo tee /etc/apt/sources.list.d/anyquery.list
sudo apt update
sudo apt install anyquery
```

### YUM/DNF

```bash
echo "[anyquery]
name=Anyquery
baseurl=https://yum.julienc.me/
enabled=1
gpgcheck=0" | sudo tee /etc/yum.repos.d/anyquery.repo
sudo dnf install anyquery
```

### Scoop

```powershell
scoop bucket add anyquery https://github.com/julien040/anyquery-scoop
scoop install anyquery
```

### Winget

```powershell
winget install JulienCagniart.anyquery
```

### Chocolatey

```powershell
choco install anyquery
```

## Plugins

Anyquery is plugin-based, and you can install plugins to extend its functionality. You can install plugins from the [official registry](https://anyquery.dev/integrations) or create your own. Anyquery can also [load any SQLite extension](https://anyquery.dev/docs/usage/plugins#using-sqlite-extensions).

![Integrations](https://anyquery.dev/images/integrations_logo.png)

## License

Anyquery is licensed under the AGPLv3 license for the core engine. The RPC library is licensed under the MIT license so that anyone can reuse plugins in different projects.

The plugins are not subject to the AGPL license. Each plugins has its own license and the copyright is owned by the plugin author.
See the [LICENSE](https://github.com/julien040/anquery/blob/main/LICENSE.md) file for more information.

## Contributing

If you want to contribute to Anyquery, please read the [contributing guidelines](https://anyquery.dev/docs/developers/project/contributing). I currently only accept minor contributions, but I'm open to any suggestions or feedback.

You can have a brief overview of the project in the [architecture](https://anyquery.dev/docs/developers/project/architecture/) documentation.
