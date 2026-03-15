package wallets_test

import (
	"app/src/routes"
	"io"
	"net/http/httptest"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Routing & Middleware Tests
// ═══════════════════════════════════════════════════════════════════════════

// RT-01 ─ Unknown processor → HTTP 404, code "40"
func TestRouting_UnknownProcessor(t *testing.T) {
	rb := postJSON(t, "/core/unknown", map[string]string{})

	if rb.HTTPStatus != 404 {
		t.Errorf("expected HTTP 404, got=%d", rb.HTTPStatus)
	}
	assertCode(t, rb, "40")
	assertContains(t, rb, "tidak ditemukan")
}

// RT-02 ─ Unknown method for known processor → HTTP 404, code "40"
func TestRouting_UnknownMethod(t *testing.T) {
	rb := postJSON(t, "/core/wallets/unknownmethod", map[string]string{})

	if rb.HTTPStatus != 404 {
		t.Errorf("expected HTTP 404, got=%d", rb.HTTPStatus)
	}
	assertCode(t, rb, "40")
	assertContains(t, rb, "tidak ditemukan")
}

// RT-03 ─ Health endpoint → HTTP 200, body "I'm Ok"
func TestRouting_HealthEndpoint(t *testing.T) {
	a := getApp()

	// Register health route if not already registered
	// getApp() only registers POST routes, so we add GET /_health here
	a.Get("/_health", routes.Health)

	req := httptest.NewRequest("GET", "/_health", nil)
	resp, err := a.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected HTTP 200, got=%d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "I'm Ok" {
		t.Errorf("expected body=%q, got=%q", "I'm Ok", string(body))
	}
}
