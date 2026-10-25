package avatars

import "testing"

func TestShippedAvatarsAreValid(t *testing.T) {
	for _, id := range []string{"ava1", "ava2", "ava3", "ava10"} {
		if !IsValidAvatarID(id) {
			t.Errorf("%s should be valid", id)
		}
	}
	for _, id := range []string{"ava11", "../x", "ava0"} {
		if IsValidAvatarID(id) {
			t.Errorf("%s should be invalid", id)
		}
	}
}
