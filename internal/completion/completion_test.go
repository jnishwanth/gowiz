package completion_test

import (
	"strings"
	"testing"

	"wiz-tui/internal/completion"
)

func TestGenerateSupportedShells(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "BASH", "ZSH", "Fish "}
	for _, sh := range shells {
		script, err := completion.Generate(sh)
		if err != nil {
			t.Fatalf("expected no error for shell %q, got: %v", sh, err)
		}
		if script == "" {
			t.Fatalf("expected non-empty completion script for shell %q", sh)
		}
	}
}

func TestGenerateUnsupportedShell(t *testing.T) {
	_, err := completion.Generate("powershell")
	if err == nil {
		t.Fatal("expected error for unsupported shell 'powershell', got nil")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestBashScriptContent(t *testing.T) {
	script := completion.Bash()
	if !strings.Contains(script, "_gowiz_completions") {
		t.Errorf("bash script missing completion function name")
	}
	if !strings.Contains(script, "complete -F _gowiz_completions gowiz") {
		t.Errorf("bash script missing complete directive")
	}
}

func TestZshScriptContent(t *testing.T) {
	script := completion.Zsh()
	if !strings.Contains(script, "#compdef gowiz") {
		t.Errorf("zsh script missing #compdef header")
	}
	if !strings.Contains(script, "_gowiz()") {
		t.Errorf("zsh script missing completion function definition")
	}
}

func TestFishScriptContent(t *testing.T) {
	script := completion.Fish()
	if !strings.Contains(script, "complete -c gowiz") {
		t.Errorf("fish script missing complete -c gowiz directive")
	}
}
