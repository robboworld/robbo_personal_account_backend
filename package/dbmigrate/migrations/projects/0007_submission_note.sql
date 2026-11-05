-- What a student writes to the teacher when handing work in: for a Scratch copy and for a
-- file submission alike. It is replaced on every hand-in and may stay empty.
ALTER TABLE scratch_projects
  ADD COLUMN IF NOT EXISTS submission_note TEXT NOT NULL DEFAULT '';

ALTER TABLE class_file_submissions
  ADD COLUMN IF NOT EXISTS submission_note TEXT NOT NULL DEFAULT '';
