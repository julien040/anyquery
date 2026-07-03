---
title: Getting Started
description: Learn more about what Anyquery is and how to use it
---

## What is anyquery ?

Anyquery allows you to write SQL queries on pretty much any data source. It is a query engine that can be used to query data from different sources like databases, APIs, and even files. For example, you can use a Notion database or a Google Sheet as a database to store your data.

**Example**

```sql
-- List all the repositories from Cloudflare ordered by stars
SELECT * FROM github_repositories_from_user('cloudflare') ORDER BY stargazers_count DESC;

-- List all your saved tracks from Spotify
SELECT * FROM spotify_saved_tracks;

-- Insert data from a git repository into a Google Sheet
INSERT INTO google_sheets_table (name, line_added) SELECT author_name, addition FROM git_commits_diff('https://github.com/vercel/next.js.git');
```

## Installation

Thank you for trying out Anyquery! You can install it by following the instructions below:

### Quick install (macOS & Linux)

The fastest way to install Anyquery. This script detects your platform, downloads the matching binary, verifies its checksum, and places it on your `PATH` — no `sudo` required:

```bash
curl -fsSL https://anyquery.dev/install.sh | sh
```

You can customise the installation with environment variables:

- `ANYQUERY_VERSION` — install a specific version (e.g. `0.4.5`) instead of the latest.
- `ANYQUERY_INSTALL_DIR` — install into a custom directory.

To update later, re-run the same command. On Windows, use Scoop, Winget, or Chocolatey (see below).

### Ubuntu, Debian, and derivatives (apt package manager)

```bash
# Add the repository
echo "deb [trusted=yes] https://apt.julienc.me/ /" | sudo tee /etc/apt/sources.list.d/anyquery.list
# Update the package list
sudo apt update
# Install the package
sudo apt install anyquery
```

### Fedora, CentOS, and derivatives (dnf/yum package manager)

```bash
# Add the repository
echo "[anyquery]
name=Anyquery
baseurl=https://yum.julienc.me/
enabled=1
gpgcheck=0" | sudo tee /etc/yum.repos.d/anyquery.repo
# Install the package
sudo dnf install anyquery
```

<!-- ### Ubuntu, Debian, and derivatives (Snapcraft)

[![Get it from the Snap Store](https://snapcraft.io/static/images/badges/en/snap-store-black.svg)](https://snapcraft.io/anyquery)

```bash
sudo snap install anyquery
``` -->

### Arch Linux (AUR)

```bash
# Install using an AUR helper like yay
yay -S anyquery-git

# paru
paru -S anyquery-git
```

### MacOS (Homebrew)

```bash
brew install anyquery
```

### Windows (Scoop)

```bash
scoop bucket add anyquery https://github.com/julien040/anyquery-scoop
scoop install anyquery
```

### Windows (Chocolatey)

```bash
choco install anyquery
```

### Windows (Winget)

```bash
winget install JulienCagniart.anyquery
```

### Any OS (Go install)

If you have Go installed (1.26+), you can install Anyquery directly from source:

```bash
CGO_ENABLED=1 go install -tags "vtable fts5 sqlite_json sqlite_math_functions" github.com/julien040/anyquery@main
```

Anyquery depends on cgo (through go-sqlite3), so make sure a C compiler (gcc, clang, or a mingw toolchain on Windows) is available in your `PATH`. The `-tags` flag is required to enable virtual tables, full-text search, JSON and math functions used by Anyquery's core features.

The binary will be installed in `$(go env GOPATH)/bin`, so make sure this directory is in your `PATH`.
