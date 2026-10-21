-- +goose Up
ALTER TABLE users
    ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE users
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE users
    ALTER COLUMN created_at SET DEFAULT now();
UPDATE users
SET created_at = now()
WHERE created_at IS NULL;
ALTER TABLE users
    ALTER COLUMN created_at SET NOT NULL;

-- +goose Down
ALTER TABLE users
    ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE users
    ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE users
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';
ALTER TABLE users
    ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP;
