-- name: CreateUser :one
INSERT INTO users (realm, display_name, email, password_hash, pin_hash, customer_id, active, created_at)
VALUES (?, ?, ?, ?, ?, ?, TRUE, ?) RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ? AND realm = 'office';

-- name: GetDriverByName :one
SELECT * FROM users WHERE display_name = ? AND realm = 'driver';

-- name: ListUsersByRealm :many
SELECT * FROM users WHERE realm = ? AND active = TRUE ORDER BY display_name;

-- name: SetUserPinHash :exec
UPDATE users SET pin_hash = ? WHERE id = ?;

-- name: SetUserActive :exec
UPDATE users SET active = ? WHERE id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, realm, expires_at) VALUES (?, ?, ?, ?);

-- name: GetSession :one
SELECT s.id, s.user_id, s.realm, s.expires_at, u.display_name, u.active
FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < ?;
