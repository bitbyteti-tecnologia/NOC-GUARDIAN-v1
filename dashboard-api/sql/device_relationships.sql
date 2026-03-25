CREATE TABLE IF NOT EXISTS device_relationships (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  tenant_id UUID NOT NULL,
  parent_device_id UUID NOT NULL,
  child_device_id UUID NOT NULL,
  relation_type TEXT NOT NULL DEFAULT 'uplink',
  discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, parent_device_id, child_device_id)
);

CREATE INDEX IF NOT EXISTS idx_device_relationships_parent
  ON device_relationships (tenant_id, parent_device_id);

CREATE INDEX IF NOT EXISTS idx_device_relationships_child
  ON device_relationships (tenant_id, child_device_id);
