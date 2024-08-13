# Shopify plugin

Run SQL queries on your Shopify store.

## Installation

You need [Anyquery](https://github.com/julien040/anyquery) to run this plugin.

Then, install the plugin with the following command:

```bash
anyquery install shopify
```

At some point, you will be asked to provide your Shopify API key. You can find it by creating a private app. To do so, follow these steps:

- Go to your [settings](https://admin.shopify.com/settings/apps)
- Click develop apps.
- Click Allow custom app development and then allow custom app development
- Go back to [https://admin.shopify.com/settings/apps/development](https://admin.shopify.com/settings/apps/development)
- Click on Create a custom app
- Fill in the name of your app (whatever you want) and click on Create app.

Now we'll configure the scopes needed for the app. Click on Select scopes and select the following scopes:

- read_products
- read_orders
- read_customers

Once done, click on Install app. Read the warning and click on Install.
Copy the API key (reveal it once) and paste it when asked by anyquery.

You'll also need to provide the shop name. You can find it in the URL of your Shopify store. For example, if your store URL is `https://my-store.myshopify.com`, then your shop name is `my-store`.

> ⚠️ Note that anyquery might access data that is subject to the GDPR. While all anyquery processing happens locally, you should be aware of the risks associated with sharing your data with third-party services. For example when using a BI tool connected to anyquery, the data might be sent to the BI tool's servers.

## Usage

Once configured, you can run SQL queries on your Shopify store. Here are some examples:

```sql
-- Get the top paying customers of your store
SELECT email, first_name, last_name, amount_spent || amount_spent_currency as total FROM shopify_customers ORDER BY amount_spent DESC LIMIT 1;
-- Get the revenue per month
SELECT strftime('%Y-%m', created_at) as month, sum(total_price) as revenue FROM shopify_orders GROUP BY month ORDER BY month;
-- List your products and their stock
SELECT title, total_inventory FROM shopify_products;
-- List how much orders were returned
SELECT count(*) as returned_orders FROM shopify_orders WHERE return_status <> 'NO_RETURN';
-- List product variants that are out of stock
SELECT display_name, product_title FROM shopify_product_variants WHERE inventory_quantity = 0;
```

## Schema

### shopify_customers

List of customers of your store.

| Column index | Column name           | type    |
| ------------ | --------------------- | ------- |
| 0            | id                    | TEXT    |
| 1            | created_at            | TEXT    |
| 2            | updated_at            | TEXT    |
| 3            | display_name          | TEXT    |
| 4            | email                 | TEXT    |
| 5            | first_name            | TEXT    |
| 6            | last_name             | TEXT    |
| 7            | locale                | TEXT    |
| 8            | note                  | TEXT    |
| 9            | phone                 | TEXT    |
| 10           | subscription_status   | TEXT    |
| 11           | state                 | TEXT    |
| 12           | tags                  | TEXT    |
| 13           | amount_spent          | INTEGER |
| 14           | amount_spent_currency | TEXT    |
| 15           | data_sale_opt_out     | INTEGER |
| 16           | valid_email           | INTEGER |
| 17           | verified_email        | INTEGER |
| 18           | tax_exempt            | INTEGER |

### shopify_orders

List the orders of your store.

| Column index | Column name         | type    |
| ------------ | ------------------- | ------- |
| 0            | id                  | TEXT    |
| 1            | name                | TEXT    |
| 2            | financial_status    | TEXT    |
| 3            | fulfillment_status  | TEXT    |
| 4            | return_status       | TEXT    |
| 5            | consumer_email      | TEXT    |
| 6            | created_at          | TEXT    |
| 7            | processed_at        | TEXT    |
| 8            | updated_at          | TEXT    |
| 9            | unpaid              | INTEGER |
| 10           | confirmed           | INTEGER |
| 11           | fulfillable         | INTEGER |
| 12           | fully_paid          | INTEGER |
| 13           | requires_shipping   | INTEGER |
| 14           | total_weight        | REAL    |
| 15           | total_price         | REAL    |
| 16           | current_total_price | REAL    |
| 17           | total_discounts     | REAL    |

### shopify_products

List the products of your store.

| Column index | Column name       | type    |
| ------------ | ----------------- | ------- |
| 0            | id                | TEXT    |
| 1            | title             | TEXT    |
| 2            | vendor            | TEXT    |
| 3            | product_type      | TEXT    |
| 4            | created_at        | TEXT    |
| 5            | updated_at        | TEXT    |
| 6            | status            | TEXT    |
| 7            | description       | TEXT    |
| 8            | description_html  | TEXT    |
| 9            | store_url         | TEXT    |
| 10           | store_preview_url | TEXT    |
| 11           | total_inventory   | INTEGER |

### shopify_product_variants

List all the product variants of the products of your store.

| Column index | Column name               | type    |
| ------------ | ------------------------- | ------- |
| 0            | id                        | TEXT    |
| 1            | barcode                   | TEXT    |
| 2            | created_at                | TEXT    |
| 3            | updated_at                | TEXT    |
| 4            | display_name              | TEXT    |
| 5            | sku                       | TEXT    |
| 6            | title                     | TEXT    |
| 7            | available_for_sale        | INTEGER |
| 8            | inventory_quantity        | INTEGER |
| 9            | position                  | TEXT    |
| 10           | sellable_online_quantity  | TEXT    |
| 11           | product_id                | TEXT    |
| 12           | product_title             | TEXT    |
| 13           | product_vendor            | TEXT    |
| 14           | product_type              | TEXT    |
| 15           | product_created_at        | TEXT    |
| 16           | product_updated_at        | TEXT    |
| 17           | product_status            | TEXT    |
| 18           | product_description       | TEXT    |
| 19           | product_description_html  | TEXT    |
| 20           | product_store_url         | TEXT    |
| 21           | product_store_preview_url | TEXT    |
| 22           | product_total_inventory   | INTEGER |

## Known limitations

- The plugin does not support the `INSERT`, `UPDATE`, and `DELETE` operations.
- The plugin caches data for an hour. If you want to refresh the data, clear the cache by running `anyquery -q "SELECT clear_plugin_cache('shopify');"`. And then restart anyquery.
- The plugin is limited by the Shopify API rate limits. If you reach the limit, you will have to wait before running more queries. As an example, the plugin can query up to 1500 rows per second for all tables except for `shopify_product_variants` which is limited to 1000 rows per second.
- Most of the time, the plugins will query all the data from the Shopify API before filtering it. This can lead to slow queries if you have a lot of data.
