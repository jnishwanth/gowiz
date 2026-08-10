package wiz

import "testing"

func TestScenes(t *testing.T) {
	t.Run("AllScenes count", func(t *testing.T) {
		if len(AllScenes) != 32 {
			t.Errorf("expected 32 scenes, got %d", len(AllScenes))
		}
	})

	t.Run("GetSceneByID valid & invalid", func(t *testing.T) {
		s1 := GetSceneByID(2)
		if s1.Name != "Sunset" {
			t.Errorf("expected Sunset for ID 2, got %s", s1.Name)
		}

		sFallback := GetSceneByID(99)
		if sFallback.ID != 1 {
			t.Errorf("expected default scene 1 for invalid ID, got %d", sFallback.ID)
		}
	})

	t.Run("GetSceneByName exact and prefix", func(t *testing.T) {
		s, found := GetSceneByName("ocean")
		if !found || s.ID != 1 {
			t.Errorf("expected Ocean scene, got %v (found=%v)", s, found)
		}

		sPrefix, foundPrefix := GetSceneByName("fire")
		if !foundPrefix || sPrefix.ID != 4 {
			t.Errorf("expected Fireplace scene for prefix fire, got %v", sPrefix)
		}

		_, notFound := GetSceneByName("nonexistent")
		if notFound {
			t.Errorf("expected false for nonexistent scene")
		}
	})

	t.Run("FilterScenes", func(t *testing.T) {
		whiteScenes := FilterScenes("white")
		if len(whiteScenes) == 0 {
			t.Errorf("expected matching white scenes, got 0")
		}

		all := FilterScenes("")
		if len(all) != 32 {
			t.Errorf("expected all 32 scenes on empty query, got %d", len(all))
		}
	})
}
