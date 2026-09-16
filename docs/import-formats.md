# CSV import formats

Headers are case-insensitive and may appear in any order. Extra columns are ignored.
Imports never delete rows and are safe to re-run: matched rows are updated in place and
columns absent from the file keep their current values.

## customers.csv

`name` (required), `billing_address`, `delivery_address`, `contact_name`, `phone`, `email`,
`delivery_notes`, `delivery_days` (`mon;thu`), `qbo_customer_id`.
Match order: `qbo_customer_id`, then exact `name`.

## products.csv

`sku` (required), `name` (required), `category`, `sell_unit` (`lb`, `case`, `each`),
`catch_weight` (`yes`/`no`), `approx_case_weight_lb` (decimal pounds), `base_price`
(decimal dollars), `qbo_item_id`. Match on `sku`. Catch-weight products must use `case`.

## prices.csv

`customer` (qbo id or exact name), `sku`, `price` (decimal dollars).
Every row creates a price effective from `--effective` (default today).

    deepcuts import customers customers.csv
    deepcuts import products products.csv
    deepcuts import prices prices.csv --effective 2026-10-01
