package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/areasong/areaflow/internal/auth"
)

const oidcStateCookieName = "areaflow_oidc_state"
const csrfCookieName = "areaflow_csrf"

type SessionAuthenticator interface {
	AuthenticateSession(context.Context, string) (auth.Principal, error)
	ValidateCSRF(auth.Principal, string) bool
	RevokeSession(context.Context, int64, string, string) error
	UpsertOIDCIdentity(context.Context, auth.OIDCIdentity) (auth.UserRecord, error)
	EnsureBootstrapPlatformAdmin(context.Context, auth.UserRecord, string, []string) error
	CreateSession(context.Context, auth.CreateSessionOptions) (auth.CreatedSession, error)
	ListRoleBindings(context.Context, string) ([]auth.RoleBinding, error)
	GrantRole(context.Context, auth.GrantRoleOptions) (auth.RoleBinding, error)
	RevokeRole(context.Context, int64, string, string) error
	CreateToken(context.Context, auth.CreateTokenOptions) (auth.CreatedToken, error)
	ListTokens(context.Context) ([]auth.TokenRecord, error)
	RevokeToken(context.Context, string, string, string) error
}

type OIDCProvider interface {
	Begin(string) (auth.OIDCLogin, error)
	Exchange(context.Context, string, string, string) (auth.OIDCIdentity, string, error)
}

type roleBindingResponse struct {
	ID         int64      `json:"id"`
	ProjectID  int64      `json:"project_id,omitempty"`
	ProjectKey string     `json:"project_key,omitempty"`
	UserID     int64      `json:"user_id,omitempty"`
	TeamID     int64      `json:"team_id,omitempty"`
	Role       string     `json:"role"`
	Status     string     `json:"status"`
	Reason     string     `json:"reason"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (s Server) oidcLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authConfig.OIDCEnabled() || s.oidcProvider == nil {
		writeError(w, http.StatusNotFound, "OIDC login is not enabled")
		return
	}
	login, err := s.oidcProvider.Begin(r.URL.Query().Get("return_to"))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "OIDC login is unavailable")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: oidcStateCookieName, Value: login.StateCookie, Path: "/api/v1/auth/oidc/callback",
		MaxAge: 600, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, login.URL, http.StatusFound)
}

func (s Server) oidcCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.oidcProvider == nil || s.sessionAuthenticator == nil {
		writeError(w, http.StatusServiceUnavailable, "OIDC callback is unavailable")
		return
	}
	stateCookie, err := r.Cookie(oidcStateCookieName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "OIDC state cookie is required")
		return
	}
	identity, returnTo, err := s.oidcProvider.Exchange(r.Context(), r.URL.Query().Get("code"), r.URL.Query().Get("state"), stateCookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "OIDC callback validation failed")
		return
	}
	user, err := s.sessionAuthenticator.UpsertOIDCIdentity(r.Context(), identity)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "persist OIDC identity failed")
		return
	}
	if err := s.sessionAuthenticator.EnsureBootstrapPlatformAdmin(r.Context(), user, identity.Subject, s.authConfig.BootstrapSubjects); err != nil {
		writeError(w, http.StatusInternalServerError, "bootstrap authorization failed")
		return
	}
	created, err := s.sessionAuthenticator.CreateSession(r.Context(), auth.CreateSessionOptions{
		User: user, Issuer: identity.Issuer, AuthTime: time.Now().UTC(),
		IdleTTL: s.authConfig.SessionIdleTTL, AbsoluteTTL: s.authConfig.SessionAbsoluteTTL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create web session failed")
		return
	}
	s.setSessionCookies(w, created)
	http.SetCookie(w, &http.Cookie{Name: oidcStateCookieName, Value: "", Path: "/api/v1/auth/oidc/callback", MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (s Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	principal := principalFromContext(r.Context())
	if principal.SessionID != 0 && s.sessionAuthenticator != nil {
		if err := s.sessionAuthenticator.RevokeSession(r.Context(), principal.SessionID, principal.Actor, "user logout"); err != nil && !errors.Is(err, auth.ErrInvalidSession) {
			writeError(w, http.StatusInternalServerError, "revoke web session failed")
			return
		}
	}
	s.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) tokensHandler(w http.ResponseWriter, r *http.Request) {
	if s.sessionAuthenticator == nil {
		writeError(w, http.StatusServiceUnavailable, "token service is unavailable")
		return
	}
	principal := principalFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		records, err := s.sessionAuthenticator.ListTokens(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list service tokens failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tokens": records})
	case http.MethodPost:
		var request struct {
			Actor        string   `json:"actor"`
			Reason       string   `json:"reason"`
			Projects     []string `json:"projects"`
			Capabilities []string `json:"capabilities"`
			ExpiresAt    string   `json:"expires_at"`
			RotatedFrom  int64    `json:"rotated_from_token_id"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid service token request")
			return
		}
		expiresAt, err := time.Parse(time.RFC3339, request.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "expires_at must be RFC3339")
			return
		}
		created, err := s.sessionAuthenticator.CreateToken(r.Context(), auth.CreateTokenOptions{
			Actor: request.Actor, CreatedBy: principal.Actor, Reason: request.Reason,
			Projects: request.Projects, Capabilities: request.Capabilities, ExpiresAt: &expiresAt, RotatedFrom: request.RotatedFrom,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, created)
	case http.MethodDelete:
		tokenKey := strings.TrimSpace(r.URL.Query().Get("token_key"))
		reason := strings.TrimSpace(r.URL.Query().Get("reason"))
		if err := s.sessionAuthenticator.RevokeToken(r.Context(), tokenKey, principal.Actor, reason); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s Server) roleBindingsHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	if s.sessionAuthenticator == nil {
		writeError(w, http.StatusServiceUnavailable, "role binding service is unavailable")
		return
	}
	principal := principalFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		bindings, err := s.sessionAuthenticator.ListRoleBindings(r.Context(), projectKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list role bindings failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"project_key": projectKey, "role_bindings": buildRoleBindingResponses(bindings)})
	case http.MethodPost:
		var request struct {
			UserID    int64  `json:"user_id"`
			TeamID    int64  `json:"team_id"`
			Role      string `json:"role"`
			Reason    string `json:"reason"`
			ExpiresAt string `json:"expires_at"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid role binding request")
			return
		}
		var expiresAt *time.Time
		if request.ExpiresAt != "" {
			parsed, err := time.Parse(time.RFC3339, request.ExpiresAt)
			if err != nil {
				writeError(w, http.StatusBadRequest, "expires_at must be RFC3339")
				return
			}
			expiresAt = &parsed
		}
		binding, err := s.sessionAuthenticator.GrantRole(r.Context(), auth.GrantRoleOptions{
			ProjectKey: projectKey, UserID: request.UserID, TeamID: request.TeamID, Role: request.Role,
			Actor: principal.Actor, Reason: request.Reason, ExpiresAt: expiresAt,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, buildRoleBindingResponse(binding))
	case http.MethodDelete:
		bindingID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("binding_id")), 10, 64)
		if err != nil || bindingID <= 0 {
			writeError(w, http.StatusBadRequest, "binding_id must be a positive integer")
			return
		}
		if err := s.sessionAuthenticator.RevokeRole(r.Context(), bindingID, principal.Actor, r.URL.Query().Get("reason")); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func buildRoleBindingResponses(bindings []auth.RoleBinding) []roleBindingResponse {
	responses := make([]roleBindingResponse, 0, len(bindings))
	for _, binding := range bindings {
		responses = append(responses, buildRoleBindingResponse(binding))
	}
	return responses
}

func buildRoleBindingResponse(binding auth.RoleBinding) roleBindingResponse {
	return roleBindingResponse{
		ID: binding.ID, ProjectID: binding.ProjectID, ProjectKey: binding.ProjectKey,
		UserID: binding.UserID, TeamID: binding.TeamID, Role: binding.Role,
		Status: binding.Status, Reason: binding.Reason, ExpiresAt: binding.ExpiresAt, CreatedAt: binding.CreatedAt,
	}
}

func (s Server) setSessionCookies(w http.ResponseWriter, created auth.CreatedSession) {
	maxAge := int(time.Until(created.ExpiresAt).Seconds())
	http.SetCookie(w, &http.Cookie{Name: s.authConfig.SessionCookieName, Value: created.RawSession, Path: "/", MaxAge: maxAge, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: created.RawCSRF, Path: "/api/", MaxAge: maxAge, Secure: true, HttpOnly: false, SameSite: http.SameSiteStrictMode})
}

func (s Server) clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: s.authConfig.SessionCookieName, Value: "", Path: "/", MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: "", Path: "/api/", MaxAge: -1, Secure: true, SameSite: http.SameSiteStrictMode})
}
