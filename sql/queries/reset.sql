-- name: DeleteUsers :exec
DELETE FROM users;

-- name: EnsureUsersHashedPassword :exec
ALTER TABLE users
ADD COLUMN IF NOT EXISTS hashed_password TEXT NOT NULL DEFAULT 'unset';