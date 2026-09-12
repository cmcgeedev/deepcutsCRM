-- name: CreateProduct :one
INSERT INTO products (sku, name, category, sell_unit, catch_weight, approx_case_weight,
  base_price_cents, qbo_item_id, active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?) RETURNING *;

-- name: UpdateProduct :one
UPDATE products SET sku = ?, name = ?, category = ?, sell_unit = ?, catch_weight = ?,
  approx_case_weight = ?, base_price_cents = ?, qbo_item_id = ?, active = ?, updated_at = ?
WHERE id = ? RETURNING *;

-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

-- name: GetProductBySKU :one
SELECT * FROM products WHERE sku = ?;

-- name: ListProducts :many
SELECT * FROM products WHERE (sqlc.arg(include_inactive) = TRUE OR active = TRUE) ORDER BY category, name;
