-- What a super admin did with someone else's content: downloads and deletions of projects and
-- "Мои файлы", always with a reason. No foreign keys: a record outlives the deleted object.
CREATE TABLE IF NOT EXISTS admin_content_audit (
  id UUID PRIMARY KEY,
  actor_user_id TEXT NOT NULL,
  target_user_id TEXT NOT NULL,
  action TEXT NOT NULL,
  object_type TEXT NOT NULL,
  object_id TEXT NOT NULL,
  object_name TEXT NOT NULL DEFAULT '',
  size_bytes BIGINT NOT NULL DEFAULT 0,
  reason TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT admin_content_audit_action CHECK (action IN ('project_download', 'project_delete', 'file_download', 'file_delete')),
  CONSTRAINT admin_content_audit_object_type CHECK (object_type IN ('project', 'file')),
  CONSTRAINT admin_content_audit_reason_len CHECK (char_length(trim(reason)) >= 3)
);

CREATE INDEX IF NOT EXISTS idx_admin_content_audit_target
  ON admin_content_audit (target_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_content_audit_created
  ON admin_content_audit (created_at DESC);
