CREATE TYPE file_access AS ENUM (
  'project_members',
  'api_key',
  'public'
);

ALTER TABLE files
  ADD COLUMN access file_access NOT NULL DEFAULT 'project_members';

CREATE INDEX files_project_id_created_at_idx ON files(project_id, created_at DESC);
