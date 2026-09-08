package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
)

// stubAnnouncementService records what reaches the service layer so tests can
// assert on the announcement the handler actually built.
type stubAnnouncementService struct {
	canManage bool
	created   *models.Announcement
}

func (s *stubAnnouncementService) CreateAnnouncement(_ context.Context, a *models.Announcement, _ string) error {
	s.created = a
	return nil
}

func (s *stubAnnouncementService) CanManage(context.Context, uuid.UUID, string) bool {
	return s.canManage
}

func newAnnouncementRig(t *testing.T, canManage bool) (*gin.Engine, *stubAnnouncementService, string) {
	t.Helper()

	base := t.TempDir()
	cfg := config.UploadConfig{
		BasePath:     base,
		MaxSizeMB:    5,
		AllowedTypes: []string{"png", "jpg", "jpeg", "gif"},
	}
	svc := &stubAnnouncementService{canManage: canManage}
	h := NewAnnouncementHandler(nil, svc, cfg)

	r := gin.New()
	r.POST("/announcements", func(c *gin.Context) {
		c.Set("employee_id", uuid.NewString())
		c.Set("role", "employee")
		c.Set("context_department_id", uuid.New())
		h.Create(c)
	})
	return r, svc, base
}

func announcementForm(t *testing.T, images map[string][]byte, existing []string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("title", "t")
	_ = w.WriteField("message", "m")
	for name, content := range images {
		part, err := w.CreateFormFile("images", name)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("write part: %v", err)
		}
	}
	if existing != nil {
		enc, _ := json.Marshal(existing)
		_ = w.WriteField("existing_images", string(enc))
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &body, w.FormDataContentType()
}

// An image posted with an announcement must land in the configured base path
// under a server-generated name, exactly like every other upload — the old
// path wrote raw client bytes into a hardcoded ./uploads/announcements the
// serving handler never reads when UPLOAD_BASE_PATH points elsewhere.
func TestAnnouncementImagesAreSanitizedAndStoredInConfiguredPath(t *testing.T) {
	r, svc, base := newAnnouncementRig(t, true)

	body, ctype := announcementForm(t, map[string][]byte{"photo.png": realPNG(t)}, nil)
	req := httptest.NewRequest(http.MethodPost, "/announcements", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if svc.created == nil || len(svc.created.Images) != 1 {
		t.Fatalf("created announcement images = %+v", svc.created)
	}
	url := svc.created.Images[0]
	if !strings.HasPrefix(url, "/api/uploads/announcements/") {
		t.Fatalf("stored URL %q does not point at the announcements store", url)
	}
	name := strings.TrimPrefix(url, "/api/uploads/announcements/")
	if strings.Contains(name, "photo") {
		t.Errorf("stored name %q retains the client filename; it must be server-generated", name)
	}
	if _, err := os.Stat(filepath.Join(base, "announcements", name)); err != nil {
		t.Errorf("stored file missing from configured base path: %v", err)
	}
}

func TestAnnouncementRejectsNonImagePayload(t *testing.T) {
	r, _, base := newAnnouncementRig(t, true)

	body, ctype := announcementForm(t, map[string][]byte{"evil.png": []byte("<html><script>alert(1)</script></html>")}, nil)
	req := httptest.NewRequest(http.MethodPost, "/announcements", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415; body %s", rec.Code, rec.Body.String())
	}
	entries, _ := os.ReadDir(filepath.Join(base, "announcements"))
	if len(entries) != 0 {
		t.Errorf("rejected payload left %d file(s) on disk", len(entries))
	}
}

// A requester who may not post announcements must be refused before any file
// is written: the old order saved images first and authorized later, leaving
// unauthorized users' files on disk.
func TestAnnouncementUnauthorizedWritesNothing(t *testing.T) {
	r, svc, base := newAnnouncementRig(t, false)

	body, ctype := announcementForm(t, map[string][]byte{"photo.png": realPNG(t)}, nil)
	req := httptest.NewRequest(http.MethodPost, "/announcements", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if svc.created != nil {
		t.Error("announcement reached the service despite failing authorization")
	}
	if entries, _ := os.ReadDir(filepath.Join(base, "announcements")); len(entries) != 0 {
		t.Errorf("unauthorized request left %d file(s) on disk", len(entries))
	}
}

// existing_images round-trips previously stored URLs through the client, so
// only paths this server could have issued may come back; anything else is an
// injection vector into <img src>.
func TestAnnouncementExistingImagesAreFiltered(t *testing.T) {
	r, svc, _ := newAnnouncementRig(t, true)

	existing := []string{
		"/api/uploads/announcements/kept.png",
		"/uploads/announcements/legacy.png",
		"https://evil.example/x.png",
		"javascript:alert(1)",
		"data:text/html,<script>1</script>",
	}
	body, ctype := announcementForm(t, nil, existing)
	req := httptest.NewRequest(http.MethodPost, "/announcements", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	want := []string{"/api/uploads/announcements/kept.png", "/uploads/announcements/legacy.png"}
	if len(svc.created.Images) != len(want) {
		t.Fatalf("images = %v, want %v", svc.created.Images, want)
	}
	for i, u := range want {
		if svc.created.Images[i] != u {
			t.Errorf("images[%d] = %q, want %q", i, svc.created.Images[i], u)
		}
	}
}

// Traversal-shaped or query-bearing strings must not survive the gate even
// with a valid prefix, and the legacy JSON path is gated identically.
func TestAnnouncementImageGateRejectsTraversalAndJSONPath(t *testing.T) {
	for _, bad := range []string{
		"/api/uploads/../../etc/passwd",
		"/uploads/images/x.png?sig=abc",
		"/api/uploads/images/x.png#frag",
		"/api/uploads/images/..\\x.png",
	} {
		if storedUploadPath(bad) {
			t.Errorf("storedUploadPath(%q) = true, want false", bad)
		}
	}
	if !storedUploadPath("/api/uploads/announcements/ab_12 photo.png") {
		t.Error("legacy filename with space was rejected")
	}

	r, svc, _ := newAnnouncementRig(t, true)
	body := `{"title":"t","message":"m","priority":"normal","images":["/api/uploads/images/ok.png","https://evil.example/x.png","/api/uploads/../secret.png"]}`
	req := httptest.NewRequest(http.MethodPost, "/announcements", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(svc.created.Images) != 1 || svc.created.Images[0] != "/api/uploads/images/ok.png" {
		t.Errorf("JSON path images = %v, want only the valid stored path", svc.created.Images)
	}
}
