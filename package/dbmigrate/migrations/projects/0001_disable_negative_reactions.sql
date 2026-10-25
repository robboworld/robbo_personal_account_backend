-- Turn off the negative reactions in the children's gallery. Existing votes stay in the
-- tables (nothing is deleted); disabled types are neither shown nor accepted, and comment
-- votes are like-only in the backend.
UPDATE scratch_reaction_types
SET is_enabled = FALSE
WHERE code IN ('dislike', 'angry', 'clown') AND is_enabled;
