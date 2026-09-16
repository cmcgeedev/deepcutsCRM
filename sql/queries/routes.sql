-- name: CreateRoute :one
INSERT INTO delivery_routes (route_date, driver_user_id, truck_label, status, created_at)
VALUES (?, ?, ?, 'planned', ?) RETURNING *;

-- name: GetRoute :one
SELECT * FROM delivery_routes WHERE id = ?;

-- name: ListRoutesByDate :many
SELECT * FROM delivery_routes WHERE route_date = ? ORDER BY id;

-- name: UpdateRouteStatus :exec
UPDATE delivery_routes SET status = ?, out_at = ?, completed_at = ? WHERE id = ?;

-- name: CreateStop :one
INSERT INTO delivery_stops (route_id, order_id, sequence, status) VALUES (?, ?, ?, 'pending') RETURNING *;

-- name: GetStop :one
SELECT * FROM delivery_stops WHERE id = ?;

-- name: ListStopsForRoute :many
SELECT s.*, o.customer_id, o.status AS order_status, o.needs_review, o.requested_delivery_date, o.notes AS order_notes,
  c.name AS customer_name, c.delivery_address, c.phone, c.contact_name, c.delivery_notes,
  CAST((SELECT count(*) FROM order_lines ol WHERE ol.order_id = o.id) AS INTEGER) AS line_count
FROM delivery_stops s
JOIN orders o ON o.id = s.order_id
JOIN customers c ON c.id = o.customer_id
WHERE s.route_id = ? ORDER BY s.sequence, s.id;

-- name: UpdateStopSequence :exec
UPDATE delivery_stops SET sequence = ? WHERE id = ? AND route_id = ?;

-- name: MaxStopSequence :one
SELECT CAST(coalesce(max(sequence), 0) AS INTEGER) FROM delivery_stops WHERE route_id = ?;

-- name: CountPendingStops :one
SELECT count(*) FROM delivery_stops WHERE route_id = ? AND status = 'pending';

-- name: UpdateStopDelivered :exec
UPDATE delivery_stops SET status = 'delivered', delivered_at = ?, proof_type = ?, proof_ref = ?, driver_note = ? WHERE id = ?;

-- name: UpdateStopSkipped :exec
UPDATE delivery_stops SET status = 'skipped', skip_reason = ?, driver_note = ? WHERE id = ?;

-- name: UpdateStopNote :exec
UPDATE delivery_stops SET driver_note = ? WHERE id = ?;

-- name: GetRouteForDriverDate :one
SELECT * FROM delivery_routes WHERE driver_user_id = ? AND route_date = ? AND status <> 'complete' ORDER BY id LIMIT 1;

-- name: CreateDriverAction :exec
INSERT INTO driver_actions (client_id, stop_id, action_type, payload, received_at) VALUES (?, ?, ?, ?, ?);

-- name: GetDriverAction :one
SELECT * FROM driver_actions WHERE client_id = ?;
