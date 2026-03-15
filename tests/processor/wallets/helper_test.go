package wallets_test

import (
	"app/src/processor/owner"
	"app/src/processor/wallets"
	"app/src/routes"
	"app/src/utils/db"
	"app/tests"
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

// ─── Shared Fiber app (singleton) ───────────────────────────────────────────

var app *fiber.App

func getApp() *fiber.App {
	if app == nil {
		app = tests.NewTestAppWithDB()

		app.Post("/core/owner", owner.Create)
		app.Post("/core/wallets", wallets.Create)
		app.Post("/core/wallets/topup", wallets.Topup)
		app.Post("/core/wallets/payment", wallets.Payment)
		app.Post("/core/wallets/transfer", wallets.Transfer)
		app.Post("/core/wallets/query", wallets.Query)

		// Dynamic routing for unknown processor/method tests
		app.Post("/core/:processor", routes.Service)
		app.Post("/core/:processor/:method", routes.Service)
	}
	return app
}

// ─── JSON helpers ───────────────────────────────────────────────────────────

func jsonBody(m map[string]string) *bytes.Buffer {
	b, _ := json.Marshal(m)
	return bytes.NewBuffer(b)
}

func postJSON(t *testing.T, url string, body map[string]string) tests.ResponseBody {
	t.Helper()
	req := httptest.NewRequest("POST", url, jsonBody(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := getApp().Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var rb tests.ResponseBody
	if err := json.Unmarshal(raw, &rb); err != nil {
		t.Fatalf("unmarshal failed: %s — raw: %s", err, string(raw))
	}
	rb.HTTPStatus = resp.StatusCode
	return rb
}

// ─── Owner helper ───────────────────────────────────────────────────────────

func createOwner(t *testing.T) string {
	t.Helper()

	rb := postJSON(t, "/core/owner", map[string]string{})
	assertCode(t, rb, "00")

	result, ok := rb.Result.(map[string]interface{})
	if !ok {
		t.Fatal("owner result not map")
	}
	return result["owner_id"].(string)
}

// ─── Wallet helpers ─────────────────────────────────────────────────────────

func createWallet(t *testing.T, ownerID, currency string) tests.ResponseBody {
	t.Helper()
	return postJSON(t, "/core/wallets", map[string]string{
		"sourceAccountNo": ownerID,
		"currency":        currency,
	})
}

func topup(t *testing.T, ownerID, currency, amount string) tests.ResponseBody {
	t.Helper()
	return postJSON(t, "/core/wallets/topup", map[string]string{
		"sourceAccountNo": ownerID,
		"currency":        currency,
		"amount":          amount,
	})
}

func payment(t *testing.T, ownerID, currency, amount string) tests.ResponseBody {
	t.Helper()
	return postJSON(t, "/core/wallets/payment", map[string]string{
		"sourceAccountNo": ownerID,
		"currency":        currency,
		"amount":          amount,
	})
}

func transfer(t *testing.T, from, to, currency, amount string) tests.ResponseBody {
	t.Helper()
	return postJSON(t, "/core/wallets/transfer", map[string]string{
		"sourceAccountNo":     from,
		"beneﬁciaryAccountNo": to,
		"currency":            currency,
		"amount":              amount,
	})
}

func queryWallet(t *testing.T, ownerID string) tests.ResponseBody {
	t.Helper()
	return postJSON(t, "/core/wallets/query", map[string]string{
		"sourceAccountNo": ownerID,
	})
}

func assertCode(t *testing.T, rb tests.ResponseBody, expected string) {
	t.Helper()
	if rb.Attribute.Code != expected {
		t.Errorf("expected code=%q, got=%q, msg=%s", expected, rb.Attribute.Code, rb.Attribute.Message)
	}
}

func assertContains(t *testing.T, rb tests.ResponseBody, substr string) {
	t.Helper()
	if !strings.Contains(strings.ToLower(rb.Attribute.Message), strings.ToLower(substr)) {
		t.Errorf("expected message to contain %q, got: %s", substr, rb.Attribute.Message)
	}
}

// ─── Result extractors ──────────────────────────────────────────────────────

func extractWallets(t *testing.T, rb tests.ResponseBody) []map[string]interface{} {
	t.Helper()
	result, ok := rb.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", rb.Result)
	}

	arr, ok := result["wallets"].([]interface{})
	if !ok {
		t.Fatalf("expected wallets array, got %T", result["wallets"])
	}

	out := make([]map[string]interface{}, len(arr))
	for i, v := range arr {
		m, ok := v.(map[string]interface{})
		if !ok {
			t.Fatalf("wallet[%d] not a map", i)
		}
		out[i] = m
	}
	return out
}

// ─── DB helpers ─────────────────────────────────────────────────────────────

func suspendWallet(t *testing.T, ownerID, currency string) {
	t.Helper()
	_, err := db.Conn().ExecContext(context.Background(),
		`UPDATE core.wallets SET status = 'SUSPENDED' WHERE owner_id = $1 AND currency = $2`,
		ownerID, currency)
	if err != nil {
		t.Fatalf("suspend wallet failed: %v", err)
	}
}

func activateWallet(t *testing.T, ownerID, currency string) {
	t.Helper()
	_, err := db.Conn().ExecContext(context.Background(),
		`UPDATE core.wallets SET status = 'ACTIVE' WHERE owner_id = $1 AND currency = $2`,
		ownerID, currency)
	if err != nil {
		t.Fatalf("activate wallet failed: %v", err)
	}
}

func getLedgerSum(t *testing.T, ownerID, currency, tipe string) string {
	t.Helper()
	var sum string
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT COALESCE(TO_CHAR(SUM(m.amount), 'FM999999999999990.00'), '0.00')
		FROM core.mutasi m
		JOIN core.wallets w ON w.wallet_id = m.wallet_id
		WHERE w.owner_id = $1 AND w.currency = $2 AND m.tipe = $3
	`, ownerID, currency, tipe).Scan(&sum)
	if err != nil {
		t.Fatalf("getLedgerSum failed: %v", err)
	}
	return sum
}

func getLedgerBalance(t *testing.T, ownerID, currency string) string {
	t.Helper()
	var balance string
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT TO_CHAR(w.balance, 'FM999999999999990.00')
		FROM core.wallets w
		WHERE w.owner_id = $1 AND w.currency = $2
	`, ownerID, currency).Scan(&balance)
	if err != nil {
		t.Fatalf("getLedgerBalance failed: %v", err)
	}
	return balance
}

func countMutasi(t *testing.T, ownerID, currency string) int {
	t.Helper()
	var count int
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT COUNT(*)
		FROM core.mutasi m
		JOIN core.wallets w ON w.wallet_id = m.wallet_id
		WHERE w.owner_id = $1 AND w.currency = $2
	`, ownerID, currency).Scan(&count)
	if err != nil {
		t.Fatalf("countMutasi failed: %v", err)
	}
	return count
}
