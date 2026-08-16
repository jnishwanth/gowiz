package styles

import (
	"testing"
)

func TestStylesColorsAndPalettes(t *testing.T) {
	// Verify color token definitions are non-empty
	if Mauve == "" || Base == "" || Yellow == "" || Green == "" || Red == "" {
		t.Errorf("Expected theme colors to be non-empty lipgloss.Color values")
	}

	// Verify panel and text styles produce rendered output
	panelOut := PanelStyle.Render("Test Panel")
	if panelOut == "" {
		t.Errorf("Expected PanelStyle.Render output to be non-empty")
	}

	activePanelOut := ActivePanelStyle.Render("Active Panel")
	if activePanelOut == "" {
		t.Errorf("Expected ActivePanelStyle.Render output to be non-empty")
	}

	titleOut := AppTitleStyle.Render("gowiz")
	if titleOut == "" {
		t.Errorf("Expected AppTitleStyle.Render output to be non-empty")
	}

	sectionOut := SectionTitleStyle.Render("Section Title")
	if sectionOut == "" {
		t.Errorf("Expected SectionTitleStyle.Render output to be non-empty")
	}

	focusedTitleOut := FocusedSectionTitleStyle.Render("Focused Section")
	if focusedTitleOut == "" {
		t.Errorf("Expected FocusedSectionTitleStyle.Render output to be non-empty")
	}

	selectedOut := SelectedItemStyle.Render("Selected")
	if selectedOut == "" {
		t.Errorf("Expected SelectedItemStyle.Render output to be non-empty")
	}

	unselectedOut := UnselectedItemStyle.Render("Unselected")
	if unselectedOut == "" {
		t.Errorf("Expected UnselectedItemStyle.Render output to be non-empty")
	}

	dimOut := DimText.Render("Dimmed text")
	if dimOut == "" {
		t.Errorf("Expected DimText.Render output to be non-empty")
	}

	statusSuccessOut := StatusSuccess.Render("Success")
	statusErrorOut := StatusError.Render("Error")
	statusWarnOut := StatusWarning.Render("Warning")
	statusInfoOut := StatusInfo.Render("Info")

	if statusSuccessOut == "" || statusErrorOut == "" || statusWarnOut == "" || statusInfoOut == "" {
		t.Errorf("Expected status style renders to be non-empty")
	}

	cmdPromptOut := CommandPromptStyle.Render(":")
	searchPromptOut := SearchPromptStyle.Render("/")
	if cmdPromptOut == "" || searchPromptOut == "" {
		t.Errorf("Expected prompt style renders to be non-empty")
	}
}
