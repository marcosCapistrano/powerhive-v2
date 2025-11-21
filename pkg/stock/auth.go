package stock

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

const (
	// DefaultUsername is the default stock firmware username.
	DefaultUsername = "root"

	// DefaultPassword is the default stock firmware password.
	DefaultPassword = "root"
)

// DigestAuth handles HTTP Digest Authentication for stock firmware.
type DigestAuth struct {
	Username string
	Password string
	nc       uint64 // nonce counter
}

// NewDigestAuth creates a new digest auth handler with default credentials.
func NewDigestAuth() *DigestAuth {
	return &DigestAuth{
		Username: DefaultUsername,
		Password: DefaultPassword,
	}
}

// NewDigestAuthWithCredentials creates a digest auth handler with custom credentials.
func NewDigestAuthWithCredentials(username, password string) *DigestAuth {
	return &DigestAuth{
		Username: username,
		Password: password,
	}
}

// DigestTransport is an http.RoundTripper that handles digest authentication.
type DigestTransport struct {
	Auth      *DigestAuth
	Transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper with digest auth support.
func (t *DigestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// First request without auth
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// If not 401, return response
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// Get WWW-Authenticate header
	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Digest ") {
		return resp, nil
	}

	// Close first response body
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Parse digest challenge
	challenge := parseDigestChallenge(authHeader)
	if challenge == nil {
		return resp, nil
	}

	// Create new request with auth
	newReq := req.Clone(req.Context())
	authValue := t.Auth.createAuthHeader(req.Method, req.URL.Path, challenge)
	newReq.Header.Set("Authorization", authValue)

	// Retry with auth
	return transport.RoundTrip(newReq)
}

// digestChallenge contains parsed WWW-Authenticate header values.
type digestChallenge struct {
	Realm     string
	Nonce     string
	QOP       string
	Algorithm string
	Opaque    string
}

// parseDigestChallenge parses a WWW-Authenticate: Digest header.
func parseDigestChallenge(header string) *digestChallenge {
	if !strings.HasPrefix(header, "Digest ") {
		return nil
	}

	challenge := &digestChallenge{}
	parts := strings.Split(header[7:], ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, "="); idx > 0 {
			key := strings.ToLower(strings.TrimSpace(part[:idx]))
			value := strings.Trim(strings.TrimSpace(part[idx+1:]), "\"")

			switch key {
			case "realm":
				challenge.Realm = value
			case "nonce":
				challenge.Nonce = value
			case "qop":
				challenge.QOP = value
			case "algorithm":
				challenge.Algorithm = value
			case "opaque":
				challenge.Opaque = value
			}
		}
	}

	return challenge
}

// createAuthHeader creates the Authorization header for digest auth.
func (a *DigestAuth) createAuthHeader(method, uri string, c *digestChallenge) string {
	// Increment nonce counter
	nc := atomic.AddUint64(&a.nc, 1)
	ncStr := fmt.Sprintf("%08x", nc)

	// Generate client nonce
	cnonce := fmt.Sprintf("%08x", nc*12345)

	// Calculate HA1 = MD5(username:realm:password)
	ha1 := md5Hash(fmt.Sprintf("%s:%s:%s", a.Username, c.Realm, a.Password))

	// Calculate HA2 = MD5(method:uri)
	ha2 := md5Hash(fmt.Sprintf("%s:%s", method, uri))

	// Calculate response
	var response string
	if c.QOP == "auth" || c.QOP == "auth-int" {
		// response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
		response = md5Hash(fmt.Sprintf("%s:%s:%s:%s:%s:%s",
			ha1, c.Nonce, ncStr, cnonce, c.QOP, ha2))
	} else {
		// response = MD5(HA1:nonce:HA2)
		response = md5Hash(fmt.Sprintf("%s:%s:%s", ha1, c.Nonce, ha2))
	}

	// Build header
	header := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		a.Username, c.Realm, c.Nonce, uri, response)

	if c.QOP != "" {
		header += fmt.Sprintf(`, qop=%s, nc=%s, cnonce="%s"`, c.QOP, ncStr, cnonce)
	}

	if c.Opaque != "" {
		header += fmt.Sprintf(`, opaque="%s"`, c.Opaque)
	}

	return header
}

// md5Hash returns the MD5 hash of a string as hex.
func md5Hash(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
