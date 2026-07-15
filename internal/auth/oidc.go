package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDCManager struct {
	issuer      string
	groupsClaim string
	oauth       oauth2.Config
	verifier    *oidc.IDTokenVerifier
	cipher      cipher.AEAD
}

type OIDCLogin struct {
	URL         string
	StateCookie string
}

type oidcState struct {
	State     string    `json:"state"`
	Nonce     string    `json:"nonce"`
	Verifier  string    `json:"verifier"`
	ReturnTo  string    `json:"return_to"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewOIDCManager(ctx context.Context, issuer, clientID, clientSecretFile, redirectURL, groupsClaim, sessionSecretFile string) (*OIDCManager, error) {
	clientSecret, err := readSecretFile(clientSecretFile)
	if err != nil {
		return nil, fmt.Errorf("read OIDC client secret: %w", err)
	}
	sessionSecret, err := readSecretFile(sessionSecretFile)
	if err != nil {
		return nil, fmt.Errorf("read session secret: %w", err)
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	key := sha256.Sum256([]byte(sessionSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if groupsClaim == "" {
		groupsClaim = "groups"
	}
	return &OIDCManager{
		issuer: issuer, groupsClaim: groupsClaim,
		oauth:    oauth2.Config{ClientID: clientID, ClientSecret: clientSecret, Endpoint: provider.Endpoint(), RedirectURL: redirectURL, Scopes: []string{oidc.ScopeOpenID, "profile", "email"}},
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}), cipher: aead,
	}, nil
}

func (m *OIDCManager) Begin(returnTo string) (OIDCLogin, error) {
	state, err := randomURLToken(32)
	if err != nil {
		return OIDCLogin{}, err
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		return OIDCLogin{}, err
	}
	verifier := oauth2.GenerateVerifier()
	payload := oidcState{State: state, Nonce: nonce, Verifier: verifier, ReturnTo: safeReturnTo(returnTo), ExpiresAt: time.Now().UTC().Add(10 * time.Minute)}
	sealed, err := m.seal(payload)
	if err != nil {
		return OIDCLogin{}, err
	}
	return OIDCLogin{
		URL:         m.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)),
		StateCookie: sealed,
	}, nil
}

func (m *OIDCManager) Exchange(ctx context.Context, code, receivedState, stateCookie string) (OIDCIdentity, string, error) {
	var state oidcState
	if err := m.open(stateCookie, &state); err != nil || time.Now().UTC().After(state.ExpiresAt) || receivedState != state.State {
		return OIDCIdentity{}, "", fmt.Errorf("invalid or expired OIDC state")
	}
	token, err := m.oauth.Exchange(ctx, code, oauth2.VerifierOption(state.Verifier))
	if err != nil {
		return OIDCIdentity{}, "", fmt.Errorf("exchange OIDC code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return OIDCIdentity{}, "", fmt.Errorf("OIDC response has no id_token")
	}
	idToken, err := m.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return OIDCIdentity{}, "", fmt.Errorf("verify OIDC id_token: %w", err)
	}
	if idToken.Nonce != state.Nonce {
		return OIDCIdentity{}, "", fmt.Errorf("OIDC nonce mismatch")
	}
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return OIDCIdentity{}, "", fmt.Errorf("decode OIDC claims: %w", err)
	}
	subject, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	displayName, _ := claims["name"].(string)
	groups := stringSliceClaim(claims[m.groupsClaim])
	if subject == "" {
		return OIDCIdentity{}, "", fmt.Errorf("OIDC subject is required")
	}
	return OIDCIdentity{Issuer: m.issuer, Subject: subject, Email: email, DisplayName: displayName, Groups: groups, Claims: claims}, state.ReturnTo, nil
}

func (m *OIDCManager) seal(value any) (string, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, m.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := m.cipher.Seal(nonce, nonce, plaintext, []byte("areaflow-oidc-state-v1"))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (m *OIDCManager) open(raw string, target any) error {
	sealed, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(sealed) < m.cipher.NonceSize() {
		return fmt.Errorf("invalid sealed state")
	}
	nonce := sealed[:m.cipher.NonceSize()]
	plaintext, err := m.cipher.Open(nil, nonce, sealed[m.cipher.NonceSize():], []byte("areaflow-oidc-state-v1"))
	if err != nil {
		return fmt.Errorf("open sealed state: %w", err)
	}
	return json.Unmarshal(plaintext, target)
}

func readSecretFile(path string) (string, error) {
	value, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(string(value))
	if len(secret) < 32 {
		return "", fmt.Errorf("secret must contain at least 32 bytes")
	}
	return secret, nil
}

func safeReturnTo(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") || strings.ContainsAny(value, "\r\n") {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") {
		return "/"
	}
	return value
}

func stringSliceClaim(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if stringsValue, ok := value.([]string); ok {
			return normalize(stringsValue)
		}
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return normalize(result)
}
