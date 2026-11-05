-- Teachers who run a class next to its owner for a limited time. Their role stays Teacher: the
-- right to the class lives here and is checked on every request, so ending it takes effect at
-- once. A row stops counting at expires_at by itself; revoked_at closes it early, or equals
-- expires_at once the same teacher is appointed again. Rows stay as the record of who had
-- access to the class. teacher_id is the LMS auth_user id, like owner_teacher_id.
CREATE TABLE IF NOT EXISTS class_co_teachers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_id UUID NOT NULL REFERENCES class_invites (id) ON DELETE CASCADE,
  teacher_id TEXT NOT NULL,
  granted_by TEXT NOT NULL,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ NULL,
  revoked_by TEXT NULL,
  -- When the "term ends tomorrow" reminder went out; extending the term clears it.
  reminded_at TIMESTAMPTZ NULL,
  CONSTRAINT class_co_teachers_teacher_nonempty CHECK (length(trim(teacher_id)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_co_teachers_open
  ON class_co_teachers (class_id, teacher_id)
  WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_co_teachers_teacher
  ON class_co_teachers (teacher_id)
  WHERE revoked_at IS NULL;
