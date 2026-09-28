package content

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRealContentLoads loads the real (gitignored) catalog and checks that every
// tile image it names exists under the repo root. Skipped without content/;
// the image check is skipped without assets/.
func TestRealContentLoads(t *testing.T) {
	const root = "../.."
	if _, err := os.Stat(filepath.Join(root, "content")); err != nil {
		t.Skip("content/ not present (gitignored)")
	}
	c, err := Load(os.DirFS(filepath.Join(root, "content")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "assets")); err != nil {
		t.Skip("assets/ not present (gitignored)")
	}

	images := map[string]string{}
	for _, f := range c.Furniture {
		images["furniture "+f.ID] = f.Image
	}
	for _, m := range c.Monsters {
		images["monster "+m.ID] = m.Image
	}
	for _, tr := range c.Traps {
		images["trap "+tr.ID] = tr.Image
	}
	for what, img := range images {
		if img == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, img)); err != nil {
			t.Errorf("%s: image %s: %v", what, img, err)
		}
	}
}
