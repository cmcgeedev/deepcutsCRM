-- name: CreateCustomer :one
INSERT INTO customers (name, billing_address, delivery_address, contact_name, phone, email,
  delivery_notes, delivery_days, qbo_customer_id, active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?) RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers SET name = ?, billing_address = ?, delivery_address = ?, contact_name = ?,
  phone = ?, email = ?, delivery_notes = ?, delivery_days = ?, qbo_customer_id = ?, active = ?, updated_at = ?
WHERE id = ? RETURNING *;

-- name: GetCustomer :one
SELECT * FROM customers WHERE id = ?;

-- name: GetCustomerByQBO :one
SELECT * FROM customers WHERE qbo_customer_id = ?;

-- name: GetCustomerByName :one
SELECT * FROM customers WHERE name = ?;

-- name: ListCustomers :many
SELECT * FROM customers WHERE (sqlc.arg(include_inactive) = TRUE OR active = TRUE) ORDER BY name;
