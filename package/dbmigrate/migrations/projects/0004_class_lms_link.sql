-- Classes no longer have to be Open edX cohorts. lms_linked = TRUE keeps the old behaviour
-- (roster in the cohort, members enrolled on the course); FALSE keeps the roster here in
-- class_members and never calls the LMS. Existing classes were all cohorts, hence the default.
-- An unlinked class has edx_course_id = '' and edx_cohort_id = 'lk:<class id>': assignments
-- and scratch_projects keep keying on that pair, and the prefix never collides with a
-- numeric edX cohort id.
ALTER TABLE class_invites
  ADD COLUMN IF NOT EXISTS lms_linked BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE class_invites DROP CONSTRAINT IF EXISTS class_invites_course_nonempty;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'class_invites_linked_course_nonempty'
  ) THEN
    ALTER TABLE class_invites
      ADD CONSTRAINT class_invites_linked_course_nonempty
      CHECK (NOT lms_linked OR length(trim(edx_course_id)) > 0);
  END IF;
END $$;

-- Roster of unlinked classes; user_id is the LMS auth_user id, like owner_teacher_id.
CREATE TABLE IF NOT EXISTS class_members (
  class_id UUID NOT NULL REFERENCES class_invites (id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (class_id, user_id),
  CONSTRAINT class_members_user_nonempty CHECK (length(trim(user_id)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_class_members_user ON class_members (user_id);
