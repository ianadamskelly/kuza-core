DROP TABLE IF EXISTS guardian_links;
DROP TABLE IF EXISTS learner_profiles;
DROP TABLE IF EXISTS content_items;
DROP TYPE IF EXISTS content_status;

ALTER TABLE organizations RENAME TO projects;
ALTER TABLE projects RENAME COLUMN kind TO template;
ALTER TABLE projects ALTER COLUMN template SET DEFAULT 'blank';
UPDATE projects SET template = 'blank' WHERE template = 'school';

CREATE TYPE project_role AS ENUM (
  'owner',
  'admin',
  'developer',
  'member'
);

ALTER TABLE memberships ALTER COLUMN role TYPE project_role
USING CASE
  WHEN role::text IN ('owner', 'admin') THEN role::text
  WHEN role::text = 'teacher' THEN 'developer'
  ELSE 'member'
END::project_role;

DROP TYPE membership_role;

ALTER TABLE memberships RENAME COLUMN organization_id TO project_id;
ALTER TABLE files RENAME COLUMN organization_id TO project_id;
ALTER TABLE audit_events RENAME COLUMN organization_id TO project_id;

CREATE TABLE project_tables (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name text NOT NULL,
  schema jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_id, name)
);

CREATE TABLE project_records (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  table_id uuid NOT NULL REFERENCES project_tables(id) ON DELETE CASCADE,
  data jsonb NOT NULL DEFAULT '{}',
  created_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX project_tables_project_id_idx ON project_tables(project_id);
CREATE INDEX project_records_project_id_table_id_idx ON project_records(project_id, table_id);
CREATE INDEX project_records_data_idx ON project_records USING gin(data);
