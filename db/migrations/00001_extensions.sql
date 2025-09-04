-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
      FROM pg_available_extensions
     WHERE name = 'uuid-ossp'
  ) THEN
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
  ELSE
    RAISE NOTICE 'extension "uuid-ossp" not available, skipping';
  END IF;
END
$$;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS "uuid-ossp";
-- +goose StatementEnd
