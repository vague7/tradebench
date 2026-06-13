package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bench/api-gateway/config"
	"github.com/bench/api-gateway/middleware"
)

// sentinel handler that records whether it was reached.
func okHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

// ── TeamAuth ─────────────────────────────────────────────────────────────────

func TestTeamAuth_MissingHeader_Returns401(t *testing.T) {
	reached := false
	handler := middleware.TeamAuth(okHandler(&reached))

	req := httptest.NewRequest(http.MethodPost, "/api/submissions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if reached {
		t.Error("inner handler must not be called when token is missing")
	}
}

func TestTeamAuth_InvalidFormat_Returns401(t *testing.T) {
	cases := []string{
		"notabearer",
		"Bearer",          // missing token value
		"Basic abc123",    // wrong scheme
		"Bearer a b",      // too many parts
	}
	for _, h := range cases {
		reached := false
		handler := middleware.TeamAuth(okHandler(&reached))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", h)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("header %q: expected 401, got %d", h, rr.Code)
		}
		if reached {
			t.Errorf("header %q: inner handler must not be called", h)
		}
	}
}

func TestTeamAuth_ValidToken_PassesThrough(t *testing.T) {
	reached := false
	handler := middleware.TeamAuth(okHandler(&reached))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer team-secret-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !reached {
		t.Error("inner handler must be called with a valid token")
	}
}

func TestTeamAuth_BearerCaseInsensitive(t *testing.T) {
	// PRD doesn't require case-insensitivity, but the implementation uses
	// strings.EqualFold — verify it works for common variants.
	for _, scheme := range []string{"bearer", "BEARER", "Bearer"} {
		reached := false
		handler := middleware.TeamAuth(okHandler(&reached))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", scheme+" mytoken")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if !reached {
			t.Errorf("scheme %q: expected pass-through, got %d", scheme, rr.Code)
		}
	}
}

// ── AdminAuth ────────────────────────────────────────────────────────────────

func TestAdminAuth_WrongToken_Returns401(t *testing.T) {
	cfg := &config.Config{AdminToken: "correct-admin-token"}
	reached := false
	handler := middleware.AdminAuth(cfg, okHandler(&reached))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if reached {
		t.Error("inner handler must not be called with wrong admin token")
	}
}

func TestAdminAuth_CorrectToken_PassesThrough(t *testing.T) {
	cfg := &config.Config{AdminToken: "correct-admin-token"}
	reached := false
	handler := middleware.AdminAuth(cfg, okHandler(&reached))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer correct-admin-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !reached {
		t.Error("inner handler must be called with correct admin token")
	}
}

func TestAdminAuth_MissingHeader_Returns401(t *testing.T) {
	cfg := &config.Config{AdminToken: "secret"}
	reached := false
	handler := middleware.AdminAuth(cfg, okHandler(&reached))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if reached {
		t.Error("inner handler must not be called with missing header")
	}
}
