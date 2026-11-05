-- Who gave the current review and when: a class can have several teachers. A new hand-in clears
-- both, like the comment. Work reviewed before this migration has NULL: nobody knows who it was.
ALTER TABLE scratch_projects
  ADD COLUMN IF NOT EXISTS reviewed_by_teacher_id TEXT NULL,
  ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ NULL;

ALTER TABLE class_file_submissions
  ADD COLUMN IF NOT EXISTS reviewed_by_teacher_id TEXT NULL,
  ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ NULL;
