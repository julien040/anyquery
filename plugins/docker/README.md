# Docker plugin

The Docker plugin allows you to query Docker containers, images and networks over SQL.

## Installation

```bash
anyquery install docker
```

## Usage

```sql
-- List all containers
SELECT * FROM docker_containers;

-- List all images from another docker daemon
SELECT * FROM docker_images('tcp://0.0.0.0:2375');

-- List all networks
SELECT * FROM docker_networks;
```

Each table can specify another docker daemon to connect to by passing the connection string as an argument to the table function. You can also set the column `host` like this:

```sql
-- List all containers from another docker daemon
SELECT * FROM docker_containers WHERE host='tcp://0.0.0.0:2375';
```

Finally, the plugin reads the environment variable `DOCKER_HOST` to connect to the docker daemon. If the variable is not set, it will connect to the default docker daemon.

## Schema

### docker_containers

| Column index | Column name  | type    |
| ------------ | ------------ | ------- |
| 0            | id           | TEXT    |
| 1            | names        | TEXT    |
| 2            | image        | TEXT    |
| 3            | image_id     | TEXT    |
| 4            | command      | TEXT    |
| 5            | created_at   | TEXT    |
| 6            | ports        | TEXT    |
| 7            | labels       | TEXT    |
| 8            | size_rw      | INTEGER |
| 9            | size_root_fs | INTEGER |
| 10           | state        | TEXT    |
| 11           | status       | TEXT    |
| 12           | networks     | TEXT    |
| 13           | mounts       | TEXT    |

### docker_container

| Column index | Column name      | type    |
| ------------ | ---------------- | ------- |
| 0            | id               | TEXT    |
| 1            | created_at       | TEXT    |
| 2            | path             | TEXT    |
| 3            | args             | TEXT    |
| 4            | container_state  | TEXT    |
| 5            | image            | TEXT    |
| 6            | resolv_conf_path | TEXT    |
| 7            | hostname_path    | TEXT    |
| 8            | hosts_path       | TEXT    |
| 9            | log_path         | TEXT    |
| 10           | name             | TEXT    |
| 11           | restart_count    | INTEGER |
| 12           | driver           | TEXT    |
| 13           | platform         | TEXT    |
| 14           | mount_label      | TEXT    |
| 15           | process_label    | TEXT    |
| 16           | host_config      | TEXT    |
| 17           | mounts           | TEXT    |
| 18           | config           | TEXT    |
| 19           | network_settings | TEXT    |

### docker_images

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | id              | TEXT    |
| 1            | created_at      | TEXT    |
| 2            | labels          | TEXT    |
| 3            | parent_id       | TEXT    |
| 4            | repo_tags       | TEXT    |
| 5            | repo_digests    | TEXT    |
| 6            | container_count | INTEGER |
| 7            | shared_size     | INTEGER |
| 8            | size            | INTEGER |

### docker_networks

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | TEXT    |
| 1            | name        | TEXT    |
| 2            | created_at  | TEXT    |
| 3            | scope       | TEXT    |
| 4            | driver      | TEXT    |
| 5            | enable_ipv6 | INTEGER |
| 6            | ipam        | TEXT    |
| 7            | containers  | TEXT    |
| 8            | options     | TEXT    |
| 9            | labels      | TEXT    |
| 10           | peers       | TEXT    |
| 11           | services    | TEXT    |
| 12           | internal    | INTEGER |
| 13           | attachable  | INTEGER |
| 14           | ingress     | INTEGER |
| 15           | config_only | INTEGER |
| 16           | config_from | TEXT    |

## Caveats

- The plugin does not do any caching. Each query will fetch the data from the docker daemon.
- The plugin does not support inserts/updates/deletes.
- Some columns are returned as JSON objects. You can use the `json_extract` or the `->>` operator to extract the data with the json path.
