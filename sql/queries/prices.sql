-- name: CreateCustomerPrice :one
INSERT INTO customer_prices (customer_id, product_id, price_cents, effective_from, created_at)
VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: ListCustomerPriceRows :many
SELECT * FROM customer_prices WHERE customer_id = ? AND product_id = ? ORDER BY effective_from DESC, id DESC;

-- name: ListCurrentCustomerPrices :many
SELECT cp.*, p.sku, p.name AS product_name
FROM customer_prices cp
JOIN products p ON p.id = cp.product_id
WHERE cp.customer_id = ?
  AND cp.id = (
    SELECT c2.id FROM customer_prices c2
    WHERE c2.customer_id = cp.customer_id AND c2.product_id = cp.product_id AND c2.effective_from <= sqlc.arg(as_of)
    ORDER BY c2.effective_from DESC, c2.id DESC LIMIT 1)
ORDER BY p.name;
