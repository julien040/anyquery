---
title: Getting Started
description: Learn more about what Anyquery is and how to use it
---

<img src="/images/docs-header.svg" alt="stars" />

## What is anyquery about ?

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

### Ubuntu, Debian, and derivatives (Snapcraft)

[![Get it from the Snap Store](https://snapcraft.io/static/images/badges/en/snap-store-black.svg)](https://snapcraft.io/anyquery)

```bash
sudo snap install anyquery
```

### MacOS (Homebrew)

```bash
brew install julien040/anyquery/anyquery
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
