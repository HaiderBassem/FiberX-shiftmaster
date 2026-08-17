package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"shiftmaster-backend/internal/upload"
)

// TokenTypeUpload marks the credential carried by the upload session cookie.
// Like every other token type it is accepted on exactly one surface, so the
// cookie is useless against the rest of the API even if it is somehow captured.
const TokenTypeUpload = "upload"

// UploadCookieName is the cookie holding the upload session credential.
const UploadCookieName = "shiftmaster_uploads"

// UploadCookiePath scopes the cookie to the upload routes. The browser will not
// attach it to any other request, so it cannot act as a general API credential.
const UploadCookiePath = "/api/uploads"

// UploadSessionTTL bounds how long the cookie remains valid. It is deliberately
// shorter than the refresh token: the SPA re-issues it on every login and
// refresh, so a live session renews it well before it lapses.
const UploadSessionTTL = 12 * time.Hour

// IssueUploadCookie sets the upload session cookie for an authenticated employee.
//
// Uploaded files are referenced by <img> tags, including tags inside rich-text
// content stored in the database. A browser does not attach Authorization
// headers to subresource requests, so header-based auth cannot protect them
// without rewriting every stored document. A cookie is the mechanism the browser
// does apply automatically, and unlike a token in the URL it never reaches an
// access log, browser history or a Referer header.
//
// HttpOnly keeps it out of reach of script, so cross-site scripting cannot
// exfiltrate it. SameSite=Strict means another origin cannot cause it to be sent,
// so a third-party page cannot embed and read protected images.
func IssueUploadCookie(c *gin.Context, claims *Claims, secret, issuer string, secure bool) error {
	token, err := signScopedToken(claims, secret, issuer, TokenTypeUpload, UploadSessionTTL)
	if err != nil {
		return err
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     UploadCookieName,
		Value:    token,
		Path:     UploadCookiePath,
		MaxAge:   int(UploadSessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

// ClearUploadCookie removes the upload session cookie.
func ClearUploadCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     UploadCookieName,
		Value:    "",
		Path:     UploadCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// UploadAccess authorises a request for a stored file.
//
// Two credentials are accepted, for two different callers:
//
//   - the upload session cookie, which the browser attaches automatically and
//     which is what makes <img> tags work, including those inside stored HTML;
//   - a signed URL, for handing a single object to something that has no cookie.
//
// Anything else is refused. Failures return 404 rather than 401 so that probing
// cannot distinguish "this file exists but you may not have it" from "no such
// file", which would otherwise confirm the existence of other people's uploads.
func UploadAccess(secret, issuer string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if signature := c.Query(upload.ParamSignature); signature != "" {
			// c.Request.URL.Path is the path actually requested, so a signature
			// issued for one object cannot be replayed against another.
			err := upload.VerifySignature(c.Request.URL.Path, secret, signature, c.Query(upload.ParamExpires))
			if err == nil {
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		cookie, err := c.Cookie(UploadCookieName)
		if err != nil || cookie == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		claims, err := ParseToken(cookie, secret, issuer, TokenTypeUpload)
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		SetAuthContext(c, claims)
		c.Next()
	}
}
