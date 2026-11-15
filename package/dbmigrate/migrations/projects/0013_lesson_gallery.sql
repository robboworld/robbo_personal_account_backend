-- The lesson gallery: files anyone in the class (students and its teachers) shares with the whole
-- class on one assignment. A row links a file in the uploader's "Мои файлы": no copy, the file
-- stays in their quota, and deleting it there takes it out of the gallery (user_files.deleted_at,
-- then the cascade when GC removes the row). Deleting the assignment removes the gallery.
-- caption, pinned and position are the teachers' edits; the uploader's file name is not touched.
CREATE TABLE IF NOT EXISTS class_gallery_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  assignment_id UUID NOT NULL REFERENCES class_assignments (id) ON DELETE CASCADE,
  user_file_id UUID NOT NULL REFERENCES user_files (id) ON DELETE CASCADE,
  uploader_user_id TEXT NOT NULL,
  caption TEXT NOT NULL DEFAULT '',
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  position INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT class_gallery_items_uploader_nonempty CHECK (length(trim(uploader_user_id)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_gallery_items_assignment_file
  ON class_gallery_items (assignment_id, user_file_id);

CREATE INDEX IF NOT EXISTS idx_class_gallery_items_file
  ON class_gallery_items (user_file_id);

-- A teacher stops students adding to the gallery (after the lesson, say); NULL means open.
-- Separate from closed_at, which stops hand-ins.
ALTER TABLE class_assignments
  ADD COLUMN IF NOT EXISTS gallery_closed_at TIMESTAMPTZ NULL;
