package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func TestREADMEStatesTheImplementedContract(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)

	for _, required := range []string{
		"go run . --http",
		"canonical item route is `/api/tasks/{id}`",
		"is not part of this contract",
		"**In-memory only.**",
		"**No authentication or authorization.**",
		"**Same-origin by default.**",
		"64 KiB",
		"No project license has been declared",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("README does not state %q", required)
		}
	}

	if strings.Contains(text, "# High-Performance Task Manager CLI") {
		t.Fatal("README retains the unsupported high-performance title")
	}
}

func TestGitignoreDoesNotImplyFilePersistence(t *testing.T) {
	gitignore, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(gitignore), "tasks.json") {
		t.Fatal(".gitignore still implies a file-backed task store")
	}
}

func TestApprovedREADMEHeaderIsUnchanged(t *testing.T) {
	asset, err := os.ReadFile(".github/assets/readme-header-dark.png")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(asset)
	if got, want := hex.EncodeToString(digest[:]), "68b65437c8d9f5164e6d88c40424ffaa06d967c342bf4023caf288a9f3c6547c"; got != want {
		t.Fatalf("README header SHA-256 = %s, want %s", got, want)
	}
}
