package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIgnoreGitPatternsApplied verifies that .gitignore entries are read and
// actually consulted by shouldIgnore. Previously the patterns were read into a
// map that nothing ever checked, so IgnoreGit was a no-op.
func TestIgnoreGitPatternsApplied(t *testing.T) {
	root := t.TempDir()
	gitignore := "generated.go\n# a comment\nbuild/\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		t.Fatal(err)
	}

	ig := Ignore{WatchedExten: []string{"*.go"}, IgnoreGit: true}
	ig.gitPatterns = readGitIgnore(root)
	if len(ig.gitPatterns) == 0 {
		t.Fatal("readGitIgnore returned no patterns")
	}

	if !ig.shouldIgnore(filepath.Join(root, "generated.go")) {
		t.Error("a .gitignore'd file was not ignored")
	}
	if ig.shouldIgnore(filepath.Join(root, "main.go")) {
		t.Error("a non-ignored file was wrongly ignored")
	}
}

func TestReadGitIgnoreMissingFile(t *testing.T) {
	if got := readGitIgnore(t.TempDir()); got != nil {
		t.Errorf("expected nil for missing .gitignore, got %v", got)
	}
}
