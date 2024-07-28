---
title: Managing Profiles
description: Learn how to have several different configurations for the same plugin
---

Profiles are a way to have several different configurations for the same plugin. For example, you can have a profile for your personal account and another one for your work account.

Table names are prefixed with the profile name unless the profile is named `default`. Let's take the example of the GitHub plugin. If you have a profile named `work`, a table will be named `work_github_my_repositories`, while the main profile will have a table named `github_my_repositories`.

You can use table aliases to avoid having to write the profile name each time you query a table. For example, you can run `SELECT * FROM my_alias` instead of `SELECT * FROM profile_github_my_repositories`.

## Create a profile

To create a profile, you can run `anyquery profiles create default <plugin-name> <profile-name>`. For example, to create a profile named `work` for the GitHub plugin, you can run:

```bash
anyquery profiles create default github work
```

You will be prompted to enter the configuration for the profile like the plugin installation. Once you have entered the configuration, the profile will be created, and you'll be able to query tables `work_github_%table_name%`.

## List profiles

To list the profiles, you can run `anyquery profiles`. It will list all the profiles you have created.

```bash
anyquery profiles
```

The list supports the same formats as the query command. You can run `anyquery profiles --format json` to get the list in JSON format.

## Update a profile

To update a profile, you can run `anyquery profiles update default <plugin-name> <profile-name>`. It will prompt you to enter the new configuration for the profile.

```bash
anyquery profiles update default github work
```

## Delete a profile

To delete a profile, you can run `anyquery profiles delete default <plugin-name> <profile-name>`. It will delete the profile and all the tables associated with it.

```bash
anyquery profiles delete default github work
```
