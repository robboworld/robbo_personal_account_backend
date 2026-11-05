-- What happened in a class and who did it, for its teachers. Rows are only added. Users are ids:
-- names stay in the LMS, and the texts of comments and messages stay out, so the history does not
-- copy personal data of children. title is the name of the object at that moment; detail depends
-- on the action, see models.ClassHistoryEventDB. No CHECK on action: a new action must not need a
-- migration, and a write that fails here is only logged.
CREATE TABLE IF NOT EXISTS class_history_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_id UUID NOT NULL REFERENCES class_invites (id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  actor_user_id TEXT NOT NULL,
  -- The role in the class at that moment: a co-teacher's term ends, the row must not change.
  actor_role TEXT NOT NULL,
  target_user_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_class_history_events_class
  ON class_history_events (class_id, created_at DESC, id DESC);
