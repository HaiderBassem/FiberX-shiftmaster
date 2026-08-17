package upload

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

const accessSecret = "5a1f3c9e7b2d48a06fc3e91b7d5a2c8e4f0b96d371ac528f"

func TestSignedURLRoundTrip(t *testing.T) {
	path := "/api/uploads/images/3f2504e0-4f89-11d3-9a0c-0305e82c3301.png"

	sig, exp := SignPath(path, accessSecret, time.Minute)
	if err := VerifySignature(path, accessSecret, sig, strconv.FormatInt(exp, 10)); err != nil {
		t.Fatalf("a freshly signed URL failed verification: %v", err)
	}
}

// A signature authorises one object. Moving it to another path must fail, or a
// single shared link would unlock the whole upload directory.
func TestSignatureIsBoundToItsPath(t *testing.T) {
	original := "/api/uploads/images/mine.png"
	other := "/api/uploads/images/someone-elses.png"

	sig, exp := SignPath(original, accessSecret, time.Minute)
	expStr := strconv.FormatInt(exp, 10)

	if err := VerifySignature(other, accessSecret, sig, expStr); err == nil {
		t.Fatal("a signature issued for one object verified against another")
	}

	// Traversal in the path must not slip past either.
	if err := VerifySignature("/api/uploads/images/../../etc/passwd", accessSecret, sig, expStr); err == nil {
		t.Fatal("a signature verified against a traversing path")
	}
}

// Extending the expiry must invalidate the signature, otherwise a link could be
// kept alive indefinitely by editing the URL.
func TestExpiryIsCoveredBySignature(t *testing.T) {
	path := "/api/uploads/images/a.png"
	sig, exp := SignPath(path, accessSecret, time.Minute)

	extended := strconv.FormatInt(exp+86_400, 10)
	if err := VerifySignature(path, accessSecret, sig, extended); err == nil {
		t.Fatal("the expiry was extended without invalidating the signature")
	}
}

// An authentically signed URL whose moment has passed must still be refused, or
// a link shared once would work forever.
func TestExpiredSignatureRejected(t *testing.T) {
	path := "/api/uploads/images/a.png"

	past := time.Now().Add(-time.Hour).Unix()
	expiredSig := signWithExpiry(path, accessSecret, past)

	if err := VerifySignature(path, accessSecret, expiredSig, strconv.FormatInt(past, 10)); err != ErrSignatureExpired {
		t.Fatalf("expired signature error = %v, want ErrSignatureExpired", err)
	}

	// A signature valid one second into the future is still accepted, proving
	// the check is on expiry rather than on the signature being old.
	soon := time.Now().Add(30 * time.Second).Unix()
	if err := VerifySignature(path, accessSecret, signWithExpiry(path, accessSecret, soon), strconv.FormatInt(soon, 10)); err != nil {
		t.Fatalf("an unexpired signature was rejected: %v", err)
	}
}

func TestTamperedSignatureRejected(t *testing.T) {
	path := "/api/uploads/images/a.png"
	sig, exp := SignPath(path, accessSecret, time.Minute)
	expStr := strconv.FormatInt(exp, 10)

	// The flipped character must actually differ from the original: a fixed
	// "A" matched the genuine signature about one run in 64 and made the test
	// flake by "tampering" into the correct value.
	flip := "A"
	if sig[0] == 'A' {
		flip = "B"
	}
	cases := map[string]string{
		"flipped character": flip + sig[1:],
		"truncated":         sig[:len(sig)-4],
		"empty":             "",
		"not base64":        "!!!not-base64!!!",
		"padded":            sig + "AAAA",
	}

	for name, tampered := range cases {
		t.Run(name, func(t *testing.T) {
			if err := VerifySignature(path, accessSecret, tampered, expStr); err == nil {
				t.Errorf("%s signature was accepted", name)
			}
		})
	}
}

func TestSignatureFromAnotherSecretRejected(t *testing.T) {
	path := "/api/uploads/images/a.png"
	sig, exp := SignPath(path, "a-completely-different-signing-secret-value", time.Minute)

	if err := VerifySignature(path, accessSecret, sig, strconv.FormatInt(exp, 10)); err == nil {
		t.Fatal("a signature made with a different secret was accepted")
	}
}

func TestMissingParametersRejected(t *testing.T) {
	path := "/api/uploads/images/a.png"
	sig, exp := SignPath(path, accessSecret, time.Minute)

	if err := VerifySignature(path, accessSecret, "", strconv.FormatInt(exp, 10)); err != ErrSignatureMissing {
		t.Error("a missing signature should be reported as missing")
	}
	if err := VerifySignature(path, accessSecret, sig, ""); err != ErrSignatureMissing {
		t.Error("a missing expiry should be reported as missing")
	}
	if err := VerifySignature(path, accessSecret, sig, "not-a-number"); err != ErrSignatureInvalid {
		t.Error("a non-numeric expiry should be reported as invalid")
	}
}

func TestBuildSignedURL(t *testing.T) {
	path := "/api/uploads/images/a.png"
	url := BuildSignedURL(path, accessSecret, time.Minute)

	if !strings.HasPrefix(url, path+"?") {
		t.Fatalf("unexpected URL shape: %s", url)
	}
	if !strings.Contains(url, ParamSignature+"=") || !strings.Contains(url, ParamExpires+"=") {
		t.Fatalf("URL is missing signature parameters: %s", url)
	}

	// The generated URL must verify against the path it names.
	query := url[strings.Index(url, "?")+1:]
	var sig, exp string
	for _, pair := range strings.Split(query, "&") {
		k, v, _ := strings.Cut(pair, "=")
		switch k {
		case ParamSignature:
			sig = v
		case ParamExpires:
			exp = v
		}
	}
	if err := VerifySignature(path, accessSecret, sig, exp); err != nil {
		t.Fatalf("generated URL does not verify: %v", err)
	}
}
