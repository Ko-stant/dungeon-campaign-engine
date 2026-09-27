package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSetsOnlyUnsetVariables(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := `# comment
DCE_TEST_PLAIN=plain value
DCE_TEST_QUOTED="quoted value"
DCE_TEST_SINGLE='single'
  DCE_TEST_SPACED = spaced
DCE_TEST_URL=postgres://u:p@localhost:5433/db?sslmode=disable
export DCE_TEST_EXPORTED=yes
DCE_TEST_PRESET=from-file

not a valid line
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DCE_TEST_PRESET", "from-environment")
	for _, k := range []string{"DCE_TEST_PLAIN", "DCE_TEST_QUOTED", "DCE_TEST_SINGLE", "DCE_TEST_SPACED", "DCE_TEST_URL", "DCE_TEST_EXPORTED"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}

	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"DCE_TEST_PLAIN":    "plain value",
		"DCE_TEST_QUOTED":   "quoted value",
		"DCE_TEST_SINGLE":   "single",
		"DCE_TEST_SPACED":   "spaced",
		"DCE_TEST_URL":      "postgres://u:p@localhost:5433/db?sslmode=disable",
		"DCE_TEST_EXPORTED": "yes",
		"DCE_TEST_PRESET":   "from-environment",
	}
	for k, v := range want {
		if got := os.Getenv(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "absent.env")); err != nil {
		t.Fatalf("missing file: %v", err)
	}
}
