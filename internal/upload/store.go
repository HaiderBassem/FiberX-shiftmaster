package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// filePerm keeps stored uploads readable by the service account only.
// Directories need the execute bit to be traversable.
const (
	filePerm os.FileMode = 0o640
	dirPerm  os.FileMode = 0o750
)

// Store writes validated image bytes into dir and returns the full path.
//
// The write is atomic: content goes to a temporary file in the same directory and
// is then renamed into place, so a crash or a full disk cannot leave a truncated
// file that later fails to decode. Callers must pass a Filename produced by
// SanitizeImage.
func Store(dir string, img *Image) (string, error) {
	if img == nil {
		return "", ErrEmptyFile
	}
	if err := ValidateStoredName(img.Filename); err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return "", fmt.Errorf("create upload directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	if _, err := tmp.Write(img.Data); err != nil {
		cleanup()
		return "", fmt.Errorf("write upload: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return "", fmt.Errorf("sync upload: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("close upload: %w", err)
	}
	if err := os.Chmod(tmpName, filePerm); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("set upload permissions: %w", err)
	}

	final := filepath.Join(dir, img.Filename)
	if err := os.Rename(tmpName, final); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("store upload: %w", err)
	}

	return final, nil
}

// ValidateStoredName rejects anything that is not a plain, single-segment filename.
//
// Names produced by SanitizeImage are always safe, so this is defence in depth for
// the retrieval path, where the name arrives from the URL. It blocks traversal
// ("../"), absolute paths, nested directories, null bytes, and the Windows
// separator, which filepath does not treat as special on Linux.
func ValidateStoredName(name string) error {
	if name == "" {
		return fmt.Errorf("empty filename")
	}
	if strings.ContainsAny(name, "/\\\x00") {
		return fmt.Errorf("invalid filename: path separators are not permitted")
	}
	if name == "." || name == ".." || strings.HasPrefix(name, ".") {
		return fmt.Errorf("invalid filename")
	}
	if name != filepath.Base(name) {
		return fmt.Errorf("invalid filename")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("invalid filename: absolute paths are not permitted")
	}
	if _, ok := ContentTypeForExt(filepath.Ext(name)); !ok {
		return fmt.Errorf("invalid filename: unsupported extension")
	}
	return nil
}

// ResolveWithin joins base and name and confirms the result stays inside base.
// It returns an error if the resolved path escapes, which also covers a symlink
// planted inside the upload directory.
func ResolveWithin(base, name string) (string, error) {
	if err := ValidateStoredName(name); err != nil {
		return "", err
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base: %w", err)
	}

	full := filepath.Join(absBase, name)
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		// The file may simply not exist; fall back to the lexical path so the
		// caller reports "not found" rather than leaking filesystem detail.
		resolved = full
	} else {
		if resolvedBase, err := filepath.EvalSymlinks(absBase); err == nil {
			absBase = resolvedBase
		}
	}

	rel, err := filepath.Rel(absBase, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid filename: path escapes the upload directory")
	}

	return full, nil
}
