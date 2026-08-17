package upload

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Signed-URL support for uploads.
//
// A signature binds one exact object path to one expiry. It is the mechanism for
// handing a single file to something that cannot present the upload session
// cookie: a link in an email, a non-browser client, an export job. In-app
// rendering does not use it, because the application embeds upload URLs inside
// stored rich-text HTML and those URLs cannot carry a signature that stays valid.
var (
	ErrSignatureMissing = errors.New("missing upload signature")
	ErrSignatureInvalid = errors.New("invalid upload signature")
	ErrSignatureExpired = errors.New("upload signature has expired")
)

// SignedURLParams are the query parameters a signed URL carries.
const (
	ParamExpires   = "exp"
	ParamSignature = "sig"
)

// signingInput is the exact string covered by the MAC.
//
// The object path is included, so a signature issued for one file cannot be
// moved to another; the expiry is included, so it cannot be extended.
func signingInput(objectPath string, expiresUnix int64) string {
	return "v1\n" + objectPath + "\n" + strconv.FormatInt(expiresUnix, 10)
}

// signWithExpiry produces the MAC for an explicit expiry.
func signWithExpiry(objectPath, secret string, expiresUnix int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput(objectPath, expiresUnix)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// SignPathAt signs objectPath for an explicit expiry, expressed as a Unix time.
// Use it when the deadline is dictated by something other than "now plus a
// duration"; SignPath is the usual entry point.
func SignPathAt(objectPath, secret string, expiresUnix int64) string {
	return signWithExpiry(objectPath, secret, expiresUnix)
}

// SignPath returns the query parameters granting temporary access to objectPath.
// objectPath must be the path the client will request, e.g. "/api/uploads/images/x.png".
//
// A non-positive ttl is treated as a caller mistake and clamped to five minutes
// rather than producing a signature that is already expired.
func SignPath(objectPath, secret string, ttl time.Duration) (signature string, expiresUnix int64) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	expiresUnix = time.Now().Add(ttl).Unix()
	return signWithExpiry(objectPath, secret, expiresUnix), expiresUnix
}

// VerifySignature checks a signature against the requested path.
func VerifySignature(objectPath, secret, signature, expires string) error {
	if signature == "" || expires == "" {
		return ErrSignatureMissing
	}

	expiresUnix, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		return ErrSignatureInvalid
	}

	// Verify the MAC before the expiry so an attacker cannot distinguish
	// "expired" from "forged" by timing or by message.
	provided, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return ErrSignatureInvalid
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput(objectPath, expiresUnix)))
	expected := mac.Sum(nil)

	// Constant time: a byte-wise comparison would leak the expected MAC.
	if !hmac.Equal(provided, expected) {
		return ErrSignatureInvalid
	}

	if time.Now().Unix() > expiresUnix {
		return ErrSignatureExpired
	}

	return nil
}

// BuildSignedURL returns objectPath with signature parameters appended.
func BuildSignedURL(objectPath, secret string, ttl time.Duration) string {
	signature, expires := SignPath(objectPath, secret, ttl)

	separator := "?"
	if strings.Contains(objectPath, "?") {
		separator = "&"
	}

	return fmt.Sprintf("%s%s%s=%d&%s=%s", objectPath, separator, ParamExpires, expires, ParamSignature, signature)
}
