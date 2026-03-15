package wallets_test

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Query / Wallet Status
// ═══════════════════════════════════════════════════════════════════════════

// W-05 ─ Query wallet SUSPENDED → menampilkan status SUSPENDED
func TestQuery_W05_SuspendedStatus(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	suspendWallet(t, ownerID, "IDR")

	q := queryWallet(t, ownerID)
	assertCode(t, q, "00")

	wallets := extractWallets(t, q)
	if len(wallets) != 1 {
		t.Fatalf("expected 1 wallet, got %d", len(wallets))
	}

	if wallets[0]["status"] != "SUSPENDED" {
		t.Errorf("expected status=SUSPENDED, got=%s", wallets[0]["status"])
	}

	// Balance tetap 100.00 (suspend tidak menghilangkan saldo)
	if wallets[0]["balance"] != "100.00" {
		t.Errorf("expected balance=100.00, got=%s", wallets[0]["balance"])
	}
}

// W-05b ─ Query multi-currency wallet → semua wallet ditampilkan
func TestQuery_W05b_MultiCurrency(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	createWallet(t, ownerID, "USD")
	createWallet(t, ownerID, "EUR")

	topup(t, ownerID, "IDR", "100.00")
	topup(t, ownerID, "USD", "50.00")

	q := queryWallet(t, ownerID)
	assertCode(t, q, "00")

	wallets := extractWallets(t, q)
	if len(wallets) != 3 {
		t.Fatalf("expected 3 wallets, got %d", len(wallets))
	}

	// Verify order: EUR, IDR, USD (alphabetical by currency)
	if wallets[0]["currency"] != "EUR" {
		t.Errorf("expected wallets[0]=EUR, got=%s", wallets[0]["currency"])
	}
	if wallets[1]["currency"] != "IDR" {
		t.Errorf("expected wallets[1]=IDR, got=%s", wallets[1]["currency"])
	}
	if wallets[2]["currency"] != "USD" {
		t.Errorf("expected wallets[2]=USD, got=%s", wallets[2]["currency"])
	}
}

// W-05c ─ Query wallet yang tidak ada → ditolak
func TestQuery_W05c_WalletNotFound(t *testing.T) {
	ownerID := createOwner(t)
	// Tidak membuat wallet

	q := queryWallet(t, ownerID)
	assertCode(t, q, "50")
	assertContains(t, q, "tidak ditemukan")
}

// W-05d ─ Query dengan field kosong → ditolak
func TestQuery_W05d_MissingField(t *testing.T) {
	rb := postJSON(t, "/core/wallets/query", map[string]string{})
	assertCode(t, rb, "40")
	assertContains(t, rb, "wajib")
}

// W-05e ─ Query menampilkan owner_id di result
func TestQuery_W05e_OwnerIDInResult(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	q := queryWallet(t, ownerID)
	assertCode(t, q, "00")

	result, ok := q.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result map")
	}

	if result["owner_id"] != ownerID {
		t.Errorf("expected owner_id=%s, got=%v", ownerID, result["owner_id"])
	}
}
