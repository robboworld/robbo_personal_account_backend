-- Who last changed the blocks of handed-in work and when: the student and the other teachers see
-- it. student_version_id is the .sb3 the student handed in: the first teacher save keeps it, or
-- retention (one version by default) would delete the child's work. A new hand-in clears all three.
ALTER TABLE scratch_projects
  ADD COLUMN IF NOT EXISTS teacher_edited_by TEXT NULL,
  ADD COLUMN IF NOT EXISTS teacher_edited_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS student_version_id UUID NULL
    REFERENCES scratch_project_versions (id) ON DELETE SET NULL;

-- ON DELETE SET NULL looks the row up on every pruned version.
CREATE INDEX IF NOT EXISTS idx_scratch_projects_student_version
  ON scratch_projects (student_version_id)
  WHERE student_version_id IS NOT NULL;
