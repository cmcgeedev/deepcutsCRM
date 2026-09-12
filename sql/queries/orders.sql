-- name: CreateOrder :one
INSERT INTO orders (customer_id, requested_delivery_date, status, notes, created_by, needs_review, created_at, updated_at)
VALUES (?, ?, 'draft', ?, ?, FALSE, ?, ?) RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders WHERE id = ?;

-- name: ListOrders :many
SELECT o.*, c.name AS customer_name,
  CAST((SELECT count(*) FROM order_lines ol WHERE ol.order_id = o.id) AS INTEGER) AS line_count
FROM orders o JOIN customers c ON c.id = o.customer_id
WHERE (sqlc.arg(date) = '' OR o.requested_delivery_date = sqlc.arg(date))
  AND (sqlc.arg(status) = '' OR o.status = sqlc.arg(status))
  AND (sqlc.arg(customer_id) = 0 OR o.customer_id = sqlc.arg(customer_id))
ORDER BY o.requested_delivery_date DESC, o.id DESC
LIMIT 500;

-- name: ListUnscheduledOrdersForDate :many
SELECT o.*, c.name AS customer_name,
  CAST((SELECT count(*) FROM order_lines ol WHERE ol.order_id = o.id) AS INTEGER) AS line_count
FROM orders o JOIN customers c ON c.id = o.customer_id
WHERE o.requested_delivery_date = ? AND o.status = 'confirmed'
ORDER BY c.name;

-- name: UpdateOrderMeta :exec
UPDATE orders SET notes = ?, requested_delivery_date = ?, updated_at = ? WHERE id = ?;

-- name: UpdateOrderStatus :exec
UPDATE orders SET status = ?, updated_at = ? WHERE id = ?;

-- name: SetOrderNeedsReview :exec
UPDATE orders SET needs_review = ?, updated_at = ? WHERE id = ?;

-- name: FinalizeOrder :exec
UPDATE orders SET status = 'finalized', needs_review = FALSE, finalized_at = ?, updated_at = ? WHERE id = ?;

-- name: CreateOrderLine :one
INSERT INTO order_lines (order_id, product_id, ordered_qty, unit_price_cents, price_overridden, est_weight)
VALUES (?, ?, ?, ?, FALSE, ?) RETURNING *;

-- name: GetOrderLine :one
SELECT * FROM order_lines WHERE id = ? AND order_id = ?;

-- name: ListOrderLines :many
SELECT ol.*, p.sku, p.name AS product_name, p.category, p.sell_unit, p.catch_weight, p.approx_case_weight, p.base_price_cents
FROM order_lines ol JOIN products p ON p.id = ol.product_id
WHERE ol.order_id = ? ORDER BY ol.id;

-- name: UpdateOrderLine :exec
UPDATE order_lines SET ordered_qty = ?, unit_price_cents = ?, price_overridden = ?, est_weight = ?,
  shipped_weight = ?, delivered_qty = ?, delivered_weight = ?, shortage_note = ?
WHERE id = ?;

-- name: DeleteOrderLine :exec
DELETE FROM order_lines WHERE id = ? AND order_id = ?;

-- name: CountOrderLines :one
SELECT count(*) FROM order_lines WHERE order_id = ?;

-- name: GetActiveStopForOrder :one
SELECT s.id AS stop_id, s.route_id, s.status AS stop_status, r.status AS route_status, r.route_date, r.driver_user_id
FROM delivery_stops s JOIN delivery_routes r ON r.id = s.route_id
WHERE s.order_id = ? AND s.status <> 'skipped';

-- name: DeleteStop :exec
DELETE FROM delivery_stops WHERE id = ?;
