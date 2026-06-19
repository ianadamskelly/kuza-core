CREATE TYPE project_table_access AS ENUM (
  'project_members',
  'api_key',
  'public'
);

ALTER TABLE project_tables
  ADD COLUMN read_access project_table_access NOT NULL DEFAULT 'project_members',
  ADD COLUMN write_access project_table_access NOT NULL DEFAULT 'project_members';

CREATE TABLE project_api_keys (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name text NOT NULL,
  token_hash text NOT NULL UNIQUE,
  token_prefix text NOT NULL,
  created_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX project_api_keys_project_id_idx ON project_api_keys(project_id);
CREATE INDEX project_api_keys_token_prefix_idx ON project_api_keys(token_prefix);
