package wiz

import "testing"

func TestScenes(t *testing.T) {
	t.Run("AllScenes count", func(t *testing.T) {
		if len(AllScenes) != 34 {
			t.Errorf("expected 34 scenes, got %d", len(AllScenes))
		}
	})

	t.Run("GetSceneByID official mapping verification", func(t *testing.T) {
		sOcean := GetSceneByID(1)
		if sOcean.Name != "Ocean" {
			t.Errorf("expected Ocean for ID 1, got %s", sOcean.Name)
		}

		sRomance := GetSceneByID(2)
		if sRomance.Name != "Romance" {
			t.Errorf("expected Romance for ID 2, got %s", sRomance.Name)
		}

		sSunset := GetSceneByID(3)
		if sSunset.Name != "Sunset" {
			t.Errorf("expected Sunset for ID 3, got %s", sSunset.Name)
		}

		sParty := GetSceneByID(4)
		if sParty.Name != "Party" {
			t.Errorf("expected Party for ID 4, got %s", sParty.Name)
		}

		sFallback := GetSceneByID(9999)
		if sFallback.ID != 1 {
			t.Errorf("expected default scene 1 for invalid ID, got %d", sFallback.ID)
		}
	})

	t.Run("GetSceneByName exact and prefix", func(t *testing.T) {
		s, found := GetSceneByName("sunset")
		if !found || s.ID != 3 {
			t.Errorf("expected Sunset scene (ID 3), got %v (found=%v)", s, found)
		}

		sPrefix, foundPrefix := GetSceneByName("rom")
		if !foundPrefix || sPrefix.ID != 2 {
			t.Errorf("expected Romance scene for prefix rom, got %v", sPrefix)
		}
	})

	t.Run("FilterScenes", func(t *testing.T) {
		whiteScenes := FilterScenes("white")
		if len(whiteScenes) == 0 {
			t.Errorf("expected matching white scenes, got 0")
		}

		all := FilterScenes("")
		if len(all) != 34 {
			t.Errorf("expected all 34 scenes on empty query, got %d", len(all))
		}
	})

	t.Run("GetSceneCategories and FilterScenesByCategory", func(t *testing.T) {
		cats := GetSceneCategories()
		if len(cats) == 0 {
			t.Fatalf("expected non-empty categories slice")
		}

		natureScenes := FilterScenesByCategory("Nature")
		if len(natureScenes) == 0 {
			t.Errorf("expected matching scenes for category Nature")
		}

		emptyCategoryScenes := FilterScenesByCategory("")
		if len(emptyCategoryScenes) != 34 {
			t.Errorf("expected 34 scenes on empty category filter, got %d", len(emptyCategoryScenes))
		}
	})

	t.Run("GetCategorySummaries", func(t *testing.T) {
		summaries := GetCategorySummaries()
		if len(summaries) == 0 {
			t.Fatalf("expected non-empty category summaries")
		}
		totalCount := 0
		for _, s := range summaries {
			if s.Category == "" {
				t.Errorf("expected non-empty category name in summary")
			}
			if s.Count != len(s.SceneIDs) || s.Count != len(s.SceneNames) {
				t.Errorf("summary count mismatch for %s: count=%d, ids=%d, names=%d", s.Category, s.Count, len(s.SceneIDs), len(s.SceneNames))
			}
			totalCount += s.Count
		}
		if totalCount != 34 {
			t.Errorf("expected total scenes across categories to be 34, got %d", totalCount)
		}
	})
}
