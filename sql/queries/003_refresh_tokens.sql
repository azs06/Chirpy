-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3,
    $4
)
RETURNING *;

-- name: DeleteRefreshTokens :exec
DELETE FROM refresh_tokens;


-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token=$1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET updated_at = NOW(), revoked_at = NOW() WHERE token=$1;
