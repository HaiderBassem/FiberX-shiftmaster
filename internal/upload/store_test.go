package upload

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Names arriving from a URL must never be able to address a file outside the
// upload directory, nor a file the serving code would hand back as a document.
func TestValidateStoredNameRejectsTraversalAndActiveContent(t *testing.T) {
	bad := []string{
		"../../etc/passwd",
		"../secret.png",
		"..%2f..%2fetc%2fpasswd.png",
		"/etc/passwd",
		"/absolute.png",
		"sub/dir/file.png",
		`..\..\windows\system32\config.png`,
		`sub\file.png`,
		"file\x00.png",
		".",
		"..",
		".hidden.png",
		"",
		"noextension",
		"payload.html",
		"payload.svg",
		"payload.js",
		"payload.php",
		"payload.png.html",
	}

	for _, name := range bad {
		t.Run(name, func(t *testing.T) {
			if err := ValidateStoredName(name); err == nil {
				t.Errorf("ValidateStoredName(%q) accepted a dangerous name", name)
			}
		})
	}

	good := []string{
		"3f2504e0-4f89-11d3-9a0c-0305e82c3301.png",
		"abc.jpg",
		"abc.jpeg",
		"abc.gif",
		"payload.html.png",
	}
	for _, name := range good {
		if err := ValidateStoredName(name); err != nil {
			t.Errorf("ValidateStoredName(%q) rejected a safe name: %v", name, err)
		}
	}
}

func TestResolveWithinBlocksEscape(t *testing.T) {
	base := t.TempDir()

	if _, err := ResolveWithin(base, "../outside.png"); err == nil {
		t.Error("traversal out of the upload directory was permitted")
	}
	if _, err := ResolveWithin(base, "/etc/passwd"); err == nil {
		t.Error("absolute path was permitted")
	}

	got, err := ResolveWithin(base, "ok.png")
	if err != nil {
		t.Fatalf("legitimate name rejected: %v", err)
	}
	if !strings.HasPrefix(got, base) {
		t.Errorf("resolved path %q escaped base %q", got, base)
	}
}

// A symlink planted inside the upload directory must not become a way to read
// arbitrary files from the host.
func TestResolveWithinBlocksSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()

	secret := filepath.Join(outside, "secret.png")
	if err := os.WriteFile(secret, []byte("sensitive"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	link := filepath.Join(base, "link.png")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := ResolveWithin(base, "link.png"); err == nil {
		t.Error("a symlink pointing outside the upload directory was resolved")
	}
}

func TestStoreWritesAtomicallyWithRestrictivePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "images")

	img, err := SanitizeImageBytes(pngBytes(t, 8, 8), defaultOpts())
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}

	path, err := Store(dir, img)
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o640 {
		t.Errorf("stored file mode = %o, want 640", perm)
	}

	// No temporary files may be left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected exactly one stored file, found %v", names)
	}
}

func TestStoreRejectsUnsafeFilename(t *testing.T) {
	dir := t.TempDir()

	img, err := SanitizeImageBytes(pngBytes(t, 8, 8), defaultOpts())
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	img.Filename = "../escape.png"

	if _, err := Store(dir, img); err == nil {
		t.Error("Store accepted a traversing filename")
	}
}
