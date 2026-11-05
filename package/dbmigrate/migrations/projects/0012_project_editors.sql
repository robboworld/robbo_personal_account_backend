-- Who has handed-in work open in the editor right now: the editor repeats its signal, and a row
-- stops counting by itself (models.EditingTTLSeconds), a closed tab sends nothing. A table of its
-- own: the signal comes every 30 seconds and must not touch scratch_projects.updated_at, which the
-- board shows as the last save. A student's draft never gets here: teachers do not see drafts.
CREATE TABLE IF NOT EXISTS scratch_project_editors (
  project_id UUID NOT NULL REFERENCES scratch_projects (id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  role TEXT NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, user_id)
);
