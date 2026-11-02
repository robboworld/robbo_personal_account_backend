-- "Мои файлы": the account's own files. The bytes live in the userFiles.storageDir blob store
-- (keyed by id); this table holds what the ЛК lists and counts against the storage quota.
-- Only the owner sees a file. deleted_at frees the quota at once; the GC worker removes the
-- row and the blob after userFiles.retainDeletedDays.
CREATE TABLE IF NOT EXISTS user_files (
  id UUID PRIMARY KEY,
  owner_user_id TEXT NOT NULL,
  original_name TEXT NOT NULL,
  kind TEXT NOT NULL,
  mime TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  sha256 CHAR(64) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ NULL,
  CONSTRAINT user_files_owner_nonempty CHECK (length(trim(owner_user_id)) > 0),
  CONSTRAINT user_files_name_len CHECK (char_length(original_name) BETWEEN 1 AND 255),
  CONSTRAINT user_files_size_nonneg CHECK (size_bytes >= 0),
  CONSTRAINT user_files_sha256_hex CHECK (sha256 ~ '^[0-9a-f]{64}$')
);

CREATE INDEX IF NOT EXISTS idx_user_files_owner_live
  ON user_files (owner_user_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_files_deleted
  ON user_files (deleted_at)
  WHERE deleted_at IS NOT NULL;
