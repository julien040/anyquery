---
title: Installing plugins
description: Learn how to install a plugin in anyquery
---

Anyquery is plugin-based, and you can install plugins to extend its functionality. You can install plugins from the [official registry](https://anyquery.dev/integrations) or create your own.

## TL;DR

<details>
<summary>How to install a plugin?</summary>

Run `anyquery plugin install <plugin-name>` in your terminal.
If you want to browse the available plugins, visit the [official registry](https://anyquery.dev/integrations) or run `anyquery plugin i` without any arguments.
</details>

## Browse the registry

You can browse the available plugins in the [official registry](https://anyquery.dev/integrations). The registry contains plugins for various saas, local apps and file formats.

![Registry](/images/docs/httpsanyquery.devintegrations_23KmyyI3@2x.png)

## Update the registry

Before running any of the operations below, you should update the registry to get the latest plugins.

```bash title="How to update the registry"
anyquery registry refresh
```

## Install a plugin

Each integration provides a tutorial on how to install it. You can install a plugin by running `anyquery install <plugin-name>` in your terminal. If the plugin requests any additional information, you will be prompted to provide it. By default, it creates a profile named `default` for the plugin. To create additional profiles (configurations), see the [profiles documentation](/docs/usage/managing-profiles).

```bash title="How to install a plugin"
anyquery install <plugin-name>
```

```bash title="Example of installing the GitHub plugin"
(base) julien@MacBook-Air-Julien ~ % anyquery install github          
✅ Successfully installed the plugin github
💪 Let's create a new profile default for the plugin github
┃ token* (type: string)                                                                                                                                   
┃ A GitHub personal access token with scopes: repo, read:org, gist, read:packages. See https://github.com/julien040/anyquery/plugins/github for more      
information.                                                                                                                                              
┃> My token
enter submit

✅ Successfully created profile default
You can now start querying these tables:
        - github_my_repositories
        - github_repositories_from_user
        - github_commits_from_repository
        - github_issues_from_repository
        - github_pull_requests_from_repository
        - github_releases_from_repository
        - github_branches_from_repository
        - github_contributors_from_repository
        - github_tags_from_repository
        - github_followers_from_user
        - github_my_followers
        - github_following_from_user
        - github_my_following
        - github_stars_from_user
        - github_my_stars
        - github_gists_from_user
        - github_my_gists
        - github_comments_from_issue
By running the following command:
        anyquery "SELECT * FROM github_my_repositories;"
You can access at anytime the list of tables by running:
        anyquery "PRAGMA table_list;"
```

## Update a plugin

To update a plugin, run `anyquery plugin update <plugin-name>` in your terminal. If the plugin has a new version, it will be downloaded and installed.

```bash title="How to update a plugin"
anyquery plugin update <plugin-name>
```

```bash title="Example of updating the GitHub plugin"
(base) julien@MacBook-Air-Julien anyquery % anyquery plugin update github        
Plugin github is already up to date
```

:::caution
Anyquery will not prompt you to update the profiles associated with the plugin. If the plugin requires a new configuration, you will need to update by running `anyquery profiles update [registry] [plugin] [profile]` (set profile and registry to `default` if you are not sure which registry and profile to use).
:::

## Remove a plugin

To remove a plugin, you need to delete all the profiles associated with it. Run `anyquery plugin remove default <plugin-name>` in your terminal. The program will normally fail and indicate which command you need to run to remove the profiles.
Once run, run again `anyquery plugin remove <plugin-name>` to remove the plugin for good.

```bash title="How to remove a plugin"
anyquery plugin remove default <plugin-name>

# Remove each profile associated with the plugin
anyquery profiles delete default <plugin-name> <profile-name>
```

```bash title="Example of removing the GitHub plugin"
(base) julien@MacBook-Air-Julien ~ % anyquery plugin rm default github     
The plugin is linked to the following profiles:
default
Please delete the profiles before uninstalling the plugin
by running the following command(s):
        anyquery profile delete default github default
(base) julien@MacBook-Air-Julien ~ % anyquery profile delete default github default
✅ Successfully deleted the profile default for the plugin github
(base) julien@MacBook-Air-Julien ~ % anyquery plugin rm default github             
✅ Successfully uninstalled the plugin github
```

## Using SQLite extensions

Anyquery can also load any SQLite extension. To do so, you need to download the extension and load it by passing the flag `--extension` and the path to anyquery. You can load multiple extensions by separating them with a comma.

```bash title="How to load many extensions"
anyquery --extension=./dist/stats.so,mod_spatialite.so
```
