# IP info with [ip-api.com](http://ip-api.com) API

This plugin provides information about an IP address using the [ip-api.com](http://ip-api.com) API.

## Setup

```bash
anyquery install ip_api
```

## Usage

```sql
-- Get information about my IP address
SELECT * FROM ip_api_info;
-- Get information about a specific IP address
SELECT * FROM ip_api_info('1.1.1.1');
```

## Schema

| Column index | Column name  | type |
| ------------ | ------------ | ---- |
| 0            | country      | TEXT |
| 1            | country_code | TEXT |
| 2            | region       | TEXT |
| 3            | region_name  | TEXT |
| 4            | city         | TEXT |
| 5            | zip          | TEXT |
| 6            | lat          | REAL |
| 7            | lon          | REAL |
| 8            | timezone     | TEXT |
| 9            | isp          | TEXT |
| 10           | org          | TEXT |
| 11           | org_as       | TEXT |
| 12           | query        | TEXT |

## Disclaimer

This plugin uses the [ip-api.com](http://ip-api.com) API to get information about an IP address. The API is free to use, but it has some limitations. Please refer to the [ip-api.com](http://ip-api.com) website for more information.
