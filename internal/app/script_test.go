package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const testScript = `# Test script

## Prologue

### P0-01 - The request

- **Speaker:** Lord Voss.

> Come in.

## Quest 1 - The Trial

### Q1-01 - The Trial begins

> The doors groan shut.

**Quest goals**
- Survive.
`

func TestCampaignScript(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }

	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Three Plagues"})
	camp := decodeAny[CampaignResponse](t, data)

	code, data := c(http.MethodGet, "/api/campaigns/"+camp.ID+"/script", nil)
	if code != http.StatusOK || decodeAny[ScriptResponse](t, data).Sections == nil {
		t.Fatalf("empty script: %d %s", code, data)
	}
	_, body := get(t, client, srv.URL+"/campaigns/"+camp.ID)
	if !strings.Contains(body, `id="script"`) || !strings.Contains(body, "No script yet") {
		t.Fatal("the campaign page should offer a script box")
	}

	resp, _ := postForm(t, client, srv.URL+"/campaigns/"+camp.ID+"/script", url.Values{"script": {testScript}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/campaigns/"+camp.ID+"#script" {
		t.Fatalf("save: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	_, data = c(http.MethodGet, "/api/campaigns/"+camp.ID+"/script", nil)
	sc := decodeAny[ScriptResponse](t, data)
	if len(sc.Sections) != 2 || sc.Sections[1].Title != "Quest 1 - The Trial" || sc.Sections[1].Passages[0].Goals[0] != "Survive." {
		t.Fatalf("parsed script: %+v", sc)
	}
	_, body = get(t, client, srv.URL+"/campaigns/"+camp.ID)
	if !strings.Contains(body, "2 passages in 2 sections") || !strings.Contains(body, "The doors groan shut.") {
		t.Fatal("the page should show the saved script and its size")
	}

	// A script that does not parse is not saved, and the error names the line.
	resp, body = postForm(t, client, srv.URL+"/campaigns/"+camp.ID+"/script", url.Values{"script": {testScript + "\nStray words.\n"}})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) || !strings.Contains(body, "line 20") {
		t.Fatalf("bad script: %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Stray words.") {
		t.Fatal("the rejected text stays in the box so it can be fixed")
	}
	_, data = c(http.MethodGet, "/api/campaigns/"+camp.ID+"/script", nil)
	if len(decodeAny[ScriptResponse](t, data).Sections) != 2 {
		t.Fatal("a rejected script must not replace the saved one")
	}

	// Clearing the box removes the script.
	postForm(t, client, srv.URL+"/campaigns/"+camp.ID+"/script", url.Values{"script": {"  "}})
	_, data = c(http.MethodGet, "/api/campaigns/"+camp.ID+"/script", nil)
	if len(decodeAny[ScriptResponse](t, data).Sections) != 0 {
		t.Fatal("cleared script")
	}

	if code, _ = c(http.MethodGet, "/api/campaigns/0190c6a0-0000-7000-8000-000000000000/script", nil); code != http.StatusNotFound {
		t.Fatalf("unknown campaign: %d", code)
	}
	if resp, _ = postForm(t, client, srv.URL+"/campaigns/0190c6a0-0000-7000-8000-000000000000/script", url.Values{"script": {testScript}}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("save to unknown campaign: %d", resp.StatusCode)
	}
}
