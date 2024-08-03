---
title: Architecture
description: Learn how Anyquery is built and how it works
---

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything. It is built on top of [SQLite](https://www.sqlite.org/index.html) and [Vitess](https://vitess.io/). Thank you for your interest in the architecture of Anyquery. This document explains how Anyquery is built and how it works.

## Core technologies

Anyquery is built on top of two main technologies: SQLite and Vitess.

### SQLite

[SQLite](https://www.sqlite.org/index.html) is a C library that provides a lightweight yet powerful SQL database engine. It has a virtual table mechanism that allows you to extend the SQL engine with custom tables. Anyquery uses this mechanism to provide tables that interact with external systems.

### Vitess

[Vitess](https://vitess.io/) is a database clustering system for horizontal scaling of MySQL. While we don't use its clustering capabilities, we use it to provide a MySQL-compatible server that can interact with the virtual tables provided by SQLite. We also leverage its SQL parsing capabilities to rewrite SQL queries on the fly so that MySQL queries can be executed on SQLite tables.

## Architecture

Here is a few diagrams that explain how Anyquery works.

### Namespace

They represent an instance of Anyquery. Each namespace has its own SQLite database (with multiple connections possible) and its own MySQL server.

![Namespace](/images/docs/Anyquery-diagram-namespace.png)

### Shell

The shell is a command-line interface that allows you to interact with Anyquery. It connects to a namespace and executes SQL queries.

![Shell](/images/docs/Anyquery-diagram-shell.png)

### MySQL server

Anyquery can act as a MySQL server, allowing you to connect to it using any MySQL client. This is useful if the shell client is not sufficient for your needs. The MySQL server connects to a [`*sql.DB`](https://pkg.go.dev/database/sql#DB) object that interacts with the SQLite database. It rewrites the SQL queries on the fly so that MySQL queries can be executed on SQLite tables.

![MySQL server](/images/docs/Anyquery-diagram-mysql.png)

## Folder structure

Here is the folder structure of Anyquery:

- `cmd/`: contains the commands declaration for [spf13/cobra](https://github.com/spf13/cobra).
- `controller/`: contains the controllers that handle each command execution.
- `module/`: contains the modules that acts as virtual tables for SQLite.
  `module/module.go` contains the virtual table for SQLite that connects to the plugins. <br>
  Other files in this folder are the virtual tables that read files (json, csv, etc.).
- `namespace/`: contains the namespace that manages the SQLite database and the MySQL server. It also features views that acts as `information_schema` and the query rewriter. Finally, it defines a few SQL functions that are not supported by SQLite.
- `other/`: contains the `prql` binding.
- `plugins/`: contains the plugins and extensions that can be downloaded from the registry. These are not subject to the Contributor License Agreement (CLA) and the AGPL license.
- `rpc/`: contains the client/server RPC communication and helpers for writing a plugin.
- `website/`: contains the website that is built with Astro and hosted on [Cloudflare Pages](https://pages.cloudflare.com/).

## Additional resources

I don't have the time to write a full documentation on the architecture of Anyquery. However, feel free to ask me any question on the GitHub discussions or by email at [contact@anyquery.dev](mailto:contact@anyquery.dev). I will be happy to answer your questions and complete this document with the information you need.
