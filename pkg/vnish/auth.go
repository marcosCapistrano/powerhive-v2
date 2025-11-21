package vnish

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	// APIKeyLength is the required length for API keys.
	APIKeyLength = 32

	// DefaultTokenTTL is the default time-to-live for cached tokens.
	// VNish doesn't specify token expiration, so we use a conservative default.
	DefaultTokenTTL = 30 * time.Minute
)

// TokenInfo holds a bearer token and its metadata.
type TokenInfo struct {
	Token     string
	ExpiresAt time.Time
}

// IsExpired returns true if the token has expired or is about to expire.
func (t *TokenInfo) IsExpired() bool {
	// Consider expired if less than 1 minute remaining
	return time.Now().Add(time.Minute).After(t.ExpiresAt)
}

// AuthManager manages authentication credentials for miners.
// It caches bearer tokens and API keys per miner (identified by host address).
type AuthManager struct {
	mu       sync.RWMutex
	tokens   map[string]*TokenInfo // host -> token info
	apiKeys  map[string]string     // host -> 32-char API key
	password string                // default unlock password
	tokenTTL time.Duration         // how long to cache tokens
}

// NewAuthManager creates a new authentication manager.
func NewAuthManager(password string) *AuthManager {
	return &AuthManager{
		tokens:   make(map[string]*TokenInfo),
		apiKeys:  make(map[string]string),
		password: password,
		tokenTTL: DefaultTokenTTL,
	}
}

// WithTokenTTL sets the token time-to-live duration.
func (am *AuthManager) WithTokenTTL(ttl time.Duration) *AuthManager {
	am.tokenTTL = ttl
	return am
}

// GetPassword returns the unlock password.
func (am *AuthManager) GetPassword() string {
	return am.password
}

// SetPassword updates the unlock password.
func (am *AuthManager) SetPassword(password string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.password = password
}

// GetToken returns the cached bearer token for a host, or empty if not cached/expired.
func (am *AuthManager) GetToken(host string) string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	info, ok := am.tokens[host]
	if !ok || info.IsExpired() {
		return ""
	}
	return info.Token
}

// SetToken caches a bearer token for a host.
func (am *AuthManager) SetToken(host, token string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.tokens[host] = &TokenInfo{
		Token:     token,
		ExpiresAt: time.Now().Add(am.tokenTTL),
	}
}

// ClearToken removes the cached token for a host.
func (am *AuthManager) ClearToken(host string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.tokens, host)
}

// GetAPIKey returns the cached API key for a host, or empty if not cached.
func (am *AuthManager) GetAPIKey(host string) string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.apiKeys[host]
}

// SetAPIKey caches an API key for a host.
func (am *AuthManager) SetAPIKey(host, apiKey string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.apiKeys[host] = apiKey
}

// ClearAPIKey removes the cached API key for a host.
func (am *AuthManager) ClearAPIKey(host string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.apiKeys, host)
}

// HasCredentials returns true if we have both token and API key for a host.
func (am *AuthManager) HasCredentials(host string) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	info, hasToken := am.tokens[host]
	_, hasKey := am.apiKeys[host]

	return hasToken && !info.IsExpired() && hasKey
}

// ClearAll removes all cached credentials for a host.
func (am *AuthManager) ClearAll(host string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.tokens, host)
	delete(am.apiKeys, host)
}

// GenerateAPIKey generates a new 32-character API key.
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, APIKeyLength/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateAPIKey checks if an API key has the correct format.
func ValidateAPIKey(key string) bool {
	return len(key) == APIKeyLength
}
