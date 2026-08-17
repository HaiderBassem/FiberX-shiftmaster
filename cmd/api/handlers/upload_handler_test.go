package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"shiftmaster-backend/internal/config"
)

func newUploadRig(t *testing.T) (*gin.Engine, string) {
	t.Helper()

	base := t.TempDir()
	cfg := config.UploadConfig{
		BasePath:     base,
		MaxSizeMB:    5,
		AllowedTypes: []string{"png", "jpg", "jpeg", "gif"},
	}
	h := NewUploadHandler(cfg)

	r := gin.New()
	r.POST("/upload/image", h.UploadImage)
	r.GET("/api/uploads/*filepath", h.ServeUpload)

	return r, base
}

func multipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &body, w.FormDataContentType()
}

func realPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func doUpload(t *testing.T, r *gin.Engine, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := multipartBody(t, "file", filename, content)
	req := httptest.NewRequest(http.MethodPost, "/upload/image", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- upload rejection -------------------------------------------------------

// The stored-XSS chain starts here: anything a browser might treat as a document
// must be refused at the door.
func TestUploadRejectsNonImageContent(t *testing.T) {
	r, _ := newUploadRig(t)

	cases := []struct {
		name     string
		filename string
		content  []byte
	}{
		{"html", "payload.html", []byte(`<html><script>alert(document.domain)</script></html>`)},
		{"html named png", "payload.png", []byte(`<html><script>alert(document.domain)</script></html>`)},
		{"svg", "payload.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		{"svg named png", "payload.png", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		{"pdf", "doc.pdf", []byte("%PDF-1.7\n1 0 obj")},
		{"empty", "empty.png", []byte{}},
		{"double extension", "payload.png.html", []byte("<html>x</html>")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doUpload(t, r, tc.filename, tc.content)
			if w.Code == http.StatusOK {
				t.Fatalf("%s was accepted (body: %s)", tc.name, w.Body.String())
			}
			if w.Code >= 500 {
				t.Errorf("expected a client error, got %d", w.Code)
			}
		})
	}
}

func TestUploadAcceptsRealImageAndGeneratesName(t *testing.T) {
	r, base := newUploadRig(t)

	w := doUpload(t, r, "../../etc/evil name.png", realPNG(t))
	if w.Code != http.StatusOK {
		t.Fatalf("legitimate PNG rejected: %d %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// The client's filename, including its traversal attempt, must not survive.
	if strings.Contains(resp.Data.URL, "evil") || strings.Contains(resp.Data.URL, "..") ||
		strings.Contains(resp.Data.URL, " ") {
		t.Fatalf("returned URL %q retains client-controlled filename content", resp.Data.URL)
	}
	if !strings.HasPrefix(resp.Data.URL, "/api/uploads/images/") {
		t.Fatalf("unexpected URL %q", resp.Data.URL)
	}

	// Exactly one file, inside the images directory.
	entries, err := os.ReadDir(filepath.Join(base, "images"))
	if err != nil {
		t.Fatalf("read images dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 stored file, got %d", len(entries))
	}
	if !strings.HasSuffix(entries[0].Name(), ".png") {
		t.Errorf("stored file %q does not have a canonical extension", entries[0].Name())
	}
}

// --- serving ----------------------------------------------------------------

func TestServeUploadPinsContentType(t *testing.T) {
	r, _ := newUploadRig(t)

	w := doUpload(t, r, "photo.png", realPNG(t))
	if w.Code != http.StatusOK {
		t.Fatalf("upload failed: %d", w.Code)
	}
	var resp struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, resp.Data.URL, nil)
	got := httptest.NewRecorder()
	r.ServeHTTP(got, req)

	if got.Code != http.StatusOK {
		t.Fatalf("serving the uploaded file failed: %d", got.Code)
	}
	if ct := got.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if got.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("nosniff header missing")
	}
	if csp := got.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
		t.Errorf("CSP = %q, want a sandbox directive", csp)
	}
}

// Files written before this validation existed are still on disk in production.
// The serving path must refuse them rather than returning them as active content.
func TestServeUploadRefusesLegacyDangerousFiles(t *testing.T) {
	r, base := newUploadRig(t)

	imagesDir := filepath.Join(base, "images")
	if err := os.MkdirAll(imagesDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	legacy := map[string]string{
		"legacy.html": `<html><script>alert(document.domain)</script></html>`,
		"legacy.svg":  `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`,
		"legacy.js":   `alert(1)`,
		"legacy.htm":  `<html>x</html>`,
	}
	for name, content := range legacy {
		if err := os.WriteFile(filepath.Join(imagesDir, name), []byte(content), 0o640); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	for name := range legacy {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/uploads/images/"+name, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404 for a legacy %s file", w.Code, filepath.Ext(name))
			}
			if ct := w.Header().Get("Content-Type"); strings.Contains(ct, "text/html") {
				t.Errorf("Content-Type = %q: legacy file served as a document", ct)
			}
		})
	}
}

func TestServeUploadBlocksTraversal(t *testing.T) {
	r, base := newUploadRig(t)

	secret := filepath.Join(filepath.Dir(base), "secret.png")
	if err := os.WriteFile(secret, []byte("sensitive"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	paths := []string{
		"/api/uploads/images/../../secret.png",
		"/api/uploads/images/..%2f..%2fsecret.png",
		"/api/uploads/../secret.png",
		"/api/uploads/images/subdir/nested.png",
		"/api/uploads/unknowndir/file.png",
		"/api/uploads/images/",
		"/api/uploads/",
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				t.Errorf("path %q was served (body: %q)", p, w.Body.String())
			}
		})
	}
}

func TestServeUploadMissingFileIsNotFound(t *testing.T) {
	r, _ := newUploadRig(t)

	req := httptest.NewRequest(http.MethodGet, "/api/uploads/images/3f2504e0-4f89-11d3-9a0c-0305e82c3301.png", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
