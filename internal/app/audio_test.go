package app

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// uploadClips posts files (name -> content) to the campaign's audio form.
func uploadClips(t *testing.T, client *http.Client, u string, files map[string]string) (*http.Response, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, content := range files {
		fw, err := mw.CreateFormFile("clips", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(fw, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	resp, err := client.Post(u, mw.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return resp, string(body)
}

type clipsResponse struct {
	Clips map[string]string `json:"clips"`
}

func TestCampaignAudio(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }

	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Three Plagues"})
	camp := decodeAny[CampaignResponse](t, data)
	postForm(t, client, srv.URL+"/campaigns/"+camp.ID+"/script", url.Values{"script": {testScript}})

	code, data := c(http.MethodGet, "/api/campaigns/"+camp.ID+"/audio", nil)
	if code != http.StatusOK || len(decodeAny[clipsResponse](t, data).Clips) != 0 {
		t.Fatalf("no clips yet: %d %s", code, data)
	}

	resp, _ := uploadClips(t, client, srv.URL+"/campaigns/"+camp.ID+"/audio", map[string]string{"P0-01.mp3": "ID3 fake mp3", "Q9-01a.ogg": "OggS"})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/campaigns/"+camp.ID+"#audio" {
		t.Fatalf("upload: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	_, data = c(http.MethodGet, "/api/campaigns/"+camp.ID+"/audio", nil)
	clips := decodeAny[clipsResponse](t, data).Clips
	if clips["P0-01"] != "/audio/"+camp.ID+"/P0-01.mp3" || clips["Q9-01a"] != "/audio/"+camp.ID+"/Q9-01a.ogg" {
		t.Fatalf("clips: %v", clips)
	}

	// Served from the database with its type; ranges (seeking) and revalidation work.
	resp, err := client.Get(srv.URL + clips["P0-01"])
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	etag := resp.Header.Get("ETag")
	if resp.StatusCode != http.StatusOK || string(body) != "ID3 fake mp3" || resp.Header.Get("Content-Type") != "audio/mpeg" || etag == "" {
		t.Fatalf("serve clip: %d %q %v", resp.StatusCode, body, resp.Header)
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+clips["P0-01"], nil)
	req.Header.Set("Range", "bytes=4-7")
	if resp, err = client.Do(req); err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent || string(body) != "fake" {
		t.Errorf("a range: %d %q", resp.StatusCode, body)
	}
	req, _ = http.NewRequest(http.MethodGet, srv.URL+clips["P0-01"], nil)
	req.Header.Set("If-None-Match", etag)
	if resp, err = client.Do(req); err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Errorf("unchanged: %d", resp.StatusCode)
	}
	// The file name must match the clip's format.
	if code, _ := get(t, client, srv.URL+"/audio/"+camp.ID+"/P0-01.ogg"); code != http.StatusNotFound {
		t.Errorf("the wrong extension: %d", code)
	}
	for _, bad := range []string{"/audio/" + camp.ID + "/nope.mp3", "/audio/" + camp.ID + "/..%2F..%2Fsecret.mp3", "/audio/not-a-campaign/P0-01.mp3"} {
		if code, _ := get(t, client, srv.URL+bad); code != http.StatusNotFound {
			t.Errorf("GET %s: %d", bad, code)
		}
	}

	_, page := get(t, client, srv.URL+"/campaigns/"+camp.ID)
	if !strings.Contains(page, `id="audio"`) || !strings.Contains(page, "P0-01.mp3") || !strings.Contains(page, "The request") ||
		!strings.Contains(page, "Q9-01a.ogg") || !strings.Contains(page, "no passage") {
		t.Fatal("the campaign page should list clips with their passages, flagging clips with no passage")
	}

	// A bad file name refuses the whole upload.
	resp, page = uploadClips(t, client, srv.URL+"/campaigns/"+camp.ID+"/audio", map[string]string{"Q1-01.mp3": "x", "notes.txt": "x"})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(page, `role="alert"`) || !strings.Contains(page, "notes.txt") {
		t.Fatalf("bad upload: %d", resp.StatusCode)
	}
	if _, data = c(http.MethodGet, "/api/campaigns/"+camp.ID+"/audio", nil); len(decodeAny[clipsResponse](t, data).Clips) != 2 {
		t.Fatal("a refused upload must not save any file")
	}

	resp, _ = postForm(t, client, srv.URL+"/campaigns/"+camp.ID+"/audio/Q9-01a/delete", url.Values{})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	if _, data = c(http.MethodGet, "/api/campaigns/"+camp.ID+"/audio", nil); len(decodeAny[clipsResponse](t, data).Clips) != 1 {
		t.Fatal("after delete")
	}
	if resp, _ = postForm(t, client, srv.URL+"/campaigns/"+camp.ID+"/audio/Q9-01a/delete", url.Values{}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete: %d", resp.StatusCode)
	}

	unknown := "/campaigns/0190c6a0-0000-7000-8000-000000000000"
	if code, _ := c(http.MethodGet, "/api"+unknown+"/audio", nil); code != http.StatusNotFound {
		t.Fatalf("unknown campaign: %d", code)
	}
	if resp, _ = uploadClips(t, client, srv.URL+unknown+"/audio", map[string]string{"P0-01.mp3": "x"}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("upload to unknown campaign: %d", resp.StatusCode)
	}
}
