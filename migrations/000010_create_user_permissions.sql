-- +goose Up
CREATE TABLE user_permissions (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id VARCHAR(80) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, permission_id)
);
CREATE INDEX ix_user_permissions_permission ON user_permissions(permission_id);

-- +goose Down
DROP TABLE user_permissions;
