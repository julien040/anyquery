# System plugin

The system plugin provides a way to query information about the system running Anyquery.

## Installation

You need [Anyquery](https://anyquery.dev/) to run this plugin.

Then, install the plugin with the following command:

```bash
anyquery install system
```

## Usage

The plugin supports only the `SELECT` statement. Here are some examples:

```sql
-- List all processes
SELECT * FROM system_processes;

-- Available memory
SELECT * FROM system_memory_stats;
```

## Tables definition

<!-- 
        "processes",
        "process_status",
        "process_memory",
        "process_files",
        "process_networks",
        "process_stats",
        // CPU related tables
        "cpu_infos",
        "cpu_stats",
        // Memory related tables
        "swaps",
        "memory_stats",
        // Disk related tables
        "partitions",
        "partition_stats",
        // Network related tables
        "network_interfaces",
        "network_stats"
 -->

### `system_processes`

List all processes running on the system.

```sql
-- List all processes
SELECT * FROM system_processes;

-- See instances of a specific executable
SELECT * FROM system_processes WHERE exe = '/usr/bin/bash';

-- See child processes of a specific process
SELECT * FROM system_processes WHERE parent_pid = 1;

-- See child processes recursively
WITH RECURSIVE children AS (
    SELECT * FROM system_processes WHERE parent_pid = 1
    UNION ALL
    SELECT p.* FROM system_processes p JOIN children c ON p.parent_pid = c.pid
)
SELECT * FROM children;

-- See processes with a specific name
SELECT * FROM system_processes WHERE name LIKE '%bash%';
```

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | pid         | INTEGER |
| 1            | name        | TEXT    |
| 2            | parent_pid  | INTEGER |
| 3            | exe         | TEXT    |
| 4            | cmdline     | TEXT    |
| 5            | cwd         | TEXT    |
| 6            | gid         | TEXT    |
| 7            | uids        | TEXT    |
| 8            | nice        | INTEGER |
| 9            | created_at  | TEXT    |

### `system_process_status`

List the status of a process. Requires the `pid` of the process. It can be either passed as an argument to the table or as a WHERE clause.

```sql
-- List the status of a process
SELECT * FROM system_process_status WHERE pid = 1;
-- Or
SELECT * FROM system_process_status(1);

-- List the status of a process with a specific name
SELECT
    p.pid,
    name,
    status
FROM
    system_processes p
    JOIN system_process_status s ON p.pid = s.pid
WHERE
    p.name LIKE '%zsh%';
```

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | status      | TEXT |

### `system_process_memory`

List the memory usage of a process. Requires the `pid` of the process. It can be either passed as an argument to the table or as a WHERE clause.

```sql
-- List the memory usage of a process
SELECT * FROM system_process_memory WHERE pid = 1;
-- Or
SELECT * FROM system_process_memory(1);

-- List the memory usage of a process with a specific name
SELECT
    p.pid,
    name,
    resident_set_size,
    virtual_memory_size
FROM
    system_processes p
    JOIN system_process_memory m ON p.pid = m.pid
WHERE
    p.name LIKE '%zsh%';
```

| Column index | Column name         | type    |
| ------------ | ------------------- | ------- |
| 0            | resident_set_size   | INTEGER |
| 1            | virtual_memory_size | INTEGER |
| 2            | high_water_mark     | INTEGER |
| 3            | data                | INTEGER |
| 4            | stack               | INTEGER |
| 5            | locked              | INTEGER |
| 6            | swap                | INTEGER |
| 7            | memory_percent      | REAL    |

### `system_process_files`

See the files opened by a process. Requires the `pid` of the process. It can be either passed as an argument to the table or as a WHERE clause.

```sql
-- List the files opened by a process
SELECT * FROM system_process_files WHERE pid = 1;
-- Or
SELECT * FROM system_process_files(1);

-- List the files opened by a process with a specific name
SELECT
    p.pid,
    name,
    path
FROM
    system_processes p
    JOIN system_process_files f ON p.pid = f.pid
WHERE
    p.name LIKE '%zsh%';
```

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | path            | TEXT    |
| 1            | file_descriptor | INTEGER |

### `system_process_networks`

See the network connections of a process. Requires the `pid` of the process. It can be either passed as an argument to the table or as a WHERE clause.

```sql
-- List the network connections of a process
SELECT * FROM system_process_networks WHERE pid = 1;
-- Or
SELECT * FROM system_process_networks(1);

-- List the network connections of a process with a specific name
SELECT
    p.pid,
    name,
    local_address,
    remote_address,
    status
FROM
    system_processes p
    JOIN system_process_networks n ON p.pid = n.pid
WHERE
    p.name LIKE '%zsh%';
```

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | file_descriptor | INTEGER |
| 1            | family          | INTEGER |
| 2            | type            | INTEGER |
| 3            | local_address   | TEXT    |
| 4            | remote_address  | TEXT    |
| 5            | status          | TEXT    |
| 6            | uid             | TEXT    |
| 7            | pid2            | INTEGER |

### `system_process_stats`

List the CPU/memory/io usage of a process. Requires the `pid` of the process. It can be either passed as an argument to the table or as a WHERE clause.

```sql
-- List the CPU/memory/io usage of a process
SELECT * FROM system_process_stats WHERE pid = 1;
-- Or
SELECT * FROM system_process_stats(1);

-- List the CPU/memory/io usage of a process with a specific name
SELECT
    p.pid,
    name,
    cpu_percent,
    memory_percent,
    io_read_count,
    io_write_count
FROM
    system_processes p
    JOIN system_process_stats s ON p.pid = s.pid
WHERE
    p.name LIKE '%zsh%';
```

| Column index | Column name       | type    |
| ------------ | ----------------- | ------- |
| 0            | cpu_affinity      | INTEGER |
| 1            | cpu_percent       | REAL    |
| 2            | memory_percent    | REAL    |
| 3            | io_read_count     | INTEGER |
| 4            | io_write_count    | INTEGER |
| 5            | io_read_bytes     | INTEGER |
| 6            | io_write_bytes    | INTEGER |
| 7            | ctx_switches      | INTEGER |
| 8            | open_files_count  | INTEGER |
| 9            | minor_page_faults | INTEGER |
| 10           | major_page_faults | INTEGER |
| 11           | cpu_user_time     | REAL    |
| 12           | cpu_system_time   | REAL    |
| 13           | cpu_idle_time     | REAL    |
| 14           | cpu_iowait_time   | REAL    |

### `system_cpu_infos`

List the CPU information. Depending of the OS, the information may return one row, or one row per core (including hyperthreading).

```sql
-- Get the CPU name
SELECT cpu_name FROM system_cpu_infos;

-- Get the count of cores
SELECT cpu_cores FROM system_cpu_infos LIMIT 1;
```

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | cpu_name        | TEXT    |
| 1            | cpu_vendor_id   | TEXT    |
| 2            | cpu_family      | TEXT    |
| 3            | cpu_model       | TEXT    |
| 4            | cpu_stepping    | INTEGER |
| 5            | cpu_physical_id | TEXT    |
| 6            | cpu_core_id     | TEXT    |
| 7            | cpu_cores       | INTEGER |
| 8            | cpu_model_name  | TEXT    |
| 9            | cpu_frequency   | INTEGER |
| 10           | cpu_cache_size  | INTEGER |
| 11           | cpu_flags       | TEXT    |
| 12           | cpu_microcode   | TEXT    |

### `system_cpu_stats`

List the CPU usage. Return one row per core (including hyperthreading).

```sql
-- Get the CPU usage
SELECT * FROM system_cpu_stats;
```

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | cpu         | TEXT |
| 1            | user        | REAL |
| 2            | system      | REAL |
| 3            | idle        | REAL |
| 4            | nice        | REAL |
| 5            | iowait      | REAL |
| 6            | irq         | REAL |
| 7            | softirq     | REAL |
| 8            | steal       | REAL |
| 9            | guest       | REAL |
| 10           | guest_nice  | REAL |

### `system_swaps`

Works only on Linux. List the swap partitions.

```sql
-- Get the swap partitions
SELECT * FROM system_swaps;

-- Get the remaining swap space
SELECT sum(free) FROM system_swaps;

-- Get the used swap space
SELECT sum(used) FROM system_swaps;

-- Get the total swap space
SELECT sum(total) FROM system_swaps;

-- Get the percentage of used swap space per swap partition
SELECT
    swap_name,
    (used * 100.0) / total AS used_percent
FROM
    system_swaps;
```

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | swap_name   | TEXT    |
| 1            | total       | INTEGER |
| 2            | used        | INTEGER |
| 3            | free        | INTEGER |

### `system_memory_stats`

List the memory usage.

```sql
-- Get the memory usage
SELECT * FROM system_memory_stats;

-- Get the total memory
SELECT total FROM system_memory_stats;

-- Get the available memory
SELECT available FROM system_memory_stats;
```

| Column index | Column name  | type    |
| ------------ | ------------ | ------- |
| 0            | total        | INTEGER |
| 1            | available    | INTEGER |
| 2            | used         | INTEGER |
| 3            | used_percent | REAL    |

### `system_partitions`

List the partitions.

```sql
-- Get the partitions
SELECT * FROM system_partitions;
```

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | device      | TEXT |
| 1            | mountpoint  | TEXT |
| 2            | fstype      | TEXT |
| 3            | options     | TEXT |

### `system_partition_stats`

List the usage of the partitions. Requires the `mountpoint` of the partition. It can be either passed as an argument to the table or as a WHERE clause.

```sql
-- Get the usage of the partitions
SELECT * FROM system_partition_stats WHERE mountpoint = '/';
-- Or
SELECT * FROM system_partition_stats('/');

-- Get the usage of the partitions with a specific device
SELECT
    device,
    total,
    free,
    used,
    used_percent
FROM
    system_partitions p
    JOIN system_partition_stats s ON p.mountpoint = s.mountpoint
WHERE
    p.device = '/dev/sda1';
```

| Column index | Column name         | type    |
| ------------ | ------------------- | ------- |
| 0            | fstype              | TEXT    |
| 1            | total               | INTEGER |
| 2            | free                | INTEGER |
| 3            | used                | INTEGER |
| 4            | used_percent        | REAL    |
| 5            | inodes_total        | INTEGER |
| 6            | inodes_used         | INTEGER |
| 7            | inodes_free         | INTEGER |
| 8            | inodes_used_percent | REAL    |

### `system_network_interfaces`

List the network interfaces.

```sql
-- Get the network interfaces
SELECT * FROM system_network_interfaces;
```

| Column index | Column name   | type |
| ------------ | ------------- | ---- |
| 0            | index         | TEXT |
| 1            | mtu           | TEXT |
| 2            | name          | TEXT |
| 3            | hardware_addr | TEXT |
| 4            | flags         | TEXT |
| 5            | addresses     | TEXT |

### `system_network_stats`

List the network statistics of each network interface.

```sql
-- Get the network statistics
SELECT * FROM system_network_stats;

-- Get the network statistics of a specific network interface
SELECT * FROM system_network_stats WHERE name = 'eth0';
```

| Column index | Column name      | type    |
| ------------ | ---------------- | ------- |
| 0            | name             | TEXT    |
| 1            | bytes_sent       | INTEGER |
| 2            | bytes_received   | INTEGER |
| 3            | packets_sent     | INTEGER |
| 4            | packets_received | INTEGER |
| 5            | err_in           | INTEGER |
| 6            | err_out          | INTEGER |
| 7            | drop_in          | INTEGER |
| 8            | drop_out         | INTEGER |
| 9            | fifo_in          | INTEGER |
| 10           | fifo_out         | INTEGER |

## Known limitations

- Some tables may not be available on all platforms.
