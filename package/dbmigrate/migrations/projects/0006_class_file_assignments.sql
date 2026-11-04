-- File assignments: students hand in files from "Мои файлы" made in any program whose files
-- the ЛК accepts (Arduino, FreeCAD, SketchUp…), not a copy of a Scratch project.
-- kind = 'scratch' keeps the old behaviour; existing assignments are all Scratch.
-- allowed_kinds are user_files.kind values; empty means any kind "Мои файлы" accepts.
-- template_file_id is the teacher's optional starter file, downloaded by the class.
ALTER TABLE class_assignments
  ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'scratch',
  ADD COLUMN IF NOT EXISTS allowed_kinds TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS template_file_id UUID NULL REFERENCES user_files (id) ON DELETE SET NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'class_assignments_kind_check'
  ) THEN
    ALTER TABLE class_assignments
      ADD CONSTRAINT class_assignments_kind_check CHECK (kind IN ('scratch', 'file'));
  END IF;
END $$;

-- One student's work on a file assignment; created when the first file is attached.
-- Statuses are those of scratch_projects.review_status.
CREATE TABLE IF NOT EXISTS class_file_submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  assignment_id UUID NOT NULL REFERENCES class_assignments (id) ON DELETE CASCADE,
  student_user_id TEXT NOT NULL,
  review_status TEXT NOT NULL DEFAULT 'draft',
  review_comment TEXT NOT NULL DEFAULT '',
  submitted_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT class_file_submissions_student_nonempty CHECK (length(trim(student_user_id)) > 0),
  CONSTRAINT class_file_submissions_review_status_check
    CHECK (review_status IN ('draft', 'submitted', 'returned', 'accepted', 'shown_in_class'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_file_submissions_assignment_student
  ON class_file_submissions (assignment_id, student_user_id);

CREATE INDEX IF NOT EXISTS idx_class_file_submissions_student
  ON class_file_submissions (student_user_id);

-- The files of a submission stay in the student's "Мои файлы" and in their quota.
CREATE TABLE IF NOT EXISTS class_submission_files (
  submission_id UUID NOT NULL REFERENCES class_file_submissions (id) ON DELETE CASCADE,
  user_file_id UUID NOT NULL REFERENCES user_files (id) ON DELETE CASCADE,
  added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (submission_id, user_file_id)
);

CREATE INDEX IF NOT EXISTS idx_class_submission_files_file
  ON class_submission_files (user_file_id);

-- When ClamAV checked the file; NULL for files uploaded before the scanner, or while it is
-- not required. A file is scanned again before it reaches a teacher or a class.
ALTER TABLE user_files
  ADD COLUMN IF NOT EXISTS scanned_at TIMESTAMPTZ NULL;
