-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- +goose StatementBegin
DO $$
BEGIN
    -- Migrate users.id from int4 to uuid if needed.
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'users'
          AND column_name = 'id'
          AND udt_name = 'int4'
    ) THEN
        ALTER TABLE users ADD COLUMN IF NOT EXISTS id_uuid UUID DEFAULT gen_random_uuid();
        UPDATE users SET id_uuid = gen_random_uuid() WHERE id_uuid IS NULL;
        ALTER TABLE users ALTER COLUMN id_uuid SET NOT NULL;

        -- If chirps.user_id is int4, add uuid column and map values.
        IF EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'chirps'
              AND column_name = 'user_id'
              AND udt_name = 'int4'
        ) THEN
            ALTER TABLE chirps ADD COLUMN IF NOT EXISTS user_id_uuid UUID;
            UPDATE chirps c
            SET user_id_uuid = u.id_uuid
            FROM users u
            WHERE c.user_id = u.id;

            ALTER TABLE chirps DROP CONSTRAINT IF EXISTS chirps_user_id_fkey;
            ALTER TABLE chirps DROP COLUMN user_id;
            ALTER TABLE chirps RENAME COLUMN user_id_uuid TO user_id;
            ALTER TABLE chirps ALTER COLUMN user_id SET NOT NULL;
        END IF;

        ALTER TABLE users DROP CONSTRAINT IF EXISTS users_pkey;
        ALTER TABLE users DROP COLUMN id;
        ALTER TABLE users RENAME COLUMN id_uuid TO id;
        ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();
        ALTER TABLE users ADD CONSTRAINT users_pkey PRIMARY KEY (id);

        IF EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'chirps'
              AND column_name = 'user_id'
              AND udt_name = 'uuid'
        ) THEN
            ALTER TABLE chirps
            ADD CONSTRAINT chirps_user_id_fkey
            FOREIGN KEY (user_id)
            REFERENCES users(id)
            ON DELETE CASCADE;
        END IF;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- Irreversible migration in practice. Retaining as no-op.
SELECT 1;
