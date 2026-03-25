CREATE TABLE IF NOT EXISTS device_relationships (
  parent_device_id uuid NOT NULL,
  child_device_id uuid NOT NULL,
  relation_type text NOT NULL DEFAULT 'connected',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_device_relationships_parent
  ON device_relationships (parent_device_id);

CREATE INDEX IF NOT EXISTS idx_device_relationships_child
  ON device_relationships (child_device_id);
