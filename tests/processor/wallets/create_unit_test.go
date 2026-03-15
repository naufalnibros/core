package wallets_test

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Wallet Management — CREATE
// ═══════════════════════════════════════════════════════════════════════════

// W-01 ─ Create wallet baru → balance = 0, status = ACTIVE
func TestCreate_W01_NewWallet(t *testing.T) {
	ownerID := createOwner(t)

	rb := createWallet(t, ownerID, "IDR")
	assertCode(t, rb, "00")
	assertContains(t, rb, "successfully")

	// Verify via query → balance 0.00, status ACTIVE
	q := queryWallet(t, ownerID)
	assertCode(t, q, "00")

	wallets := extractWallets(t, q)
	if len(wallets) != 1 {
		t.Fatalf("expected 1 wallet, got %d", len(wallets))
	}

	w := wallets[0]
	if w["currency"] != "IDR" {
		t.Errorf("expected currency=IDR, got=%s", w["currency"])
	}
	if w["balance"] != "0.00" {
		t.Errorf("expected balance=0.00, got=%s", w["balance"])
	}
	if w["status"] != "ACTIVE" {
		t.Errorf("expected status=ACTIVE, got=%s", w["status"])
	}
}

// W-02 ─ Create wallet duplikat (currency sudah ada) → ditolak
func TestCreate_W02_DuplicateCurrency(t *testing.T) {
	ownerID := createOwner(t)

	rb1 := createWallet(t, ownerID, "IDR")
	assertCode(t, rb1, "00")

	rb2 := createWallet(t, ownerID, "IDR")
	assertCode(t, rb2, "50")
	assertContains(t, rb2, "duplicate")
}

// W-03 ─ Create multi-currency per owner (IDR + USD + EUR)
func TestCreate_W03_MultiCurrency(t *testing.T) {
	ownerID := createOwner(t)

	currencies := []string{"IDR", "USD", "EUR"}
	for _, c := range currencies {
		rb := createWallet(t, ownerID, c)
		assertCode(t, rb, "00")
	}

	q := queryWallet(t, ownerID)
	assertCode(t, q, "00")

	wallets := extractWallets(t, q)
	if len(wallets) != 3 {
		t.Fatalf("expected 3 wallets, got %d", len(wallets))
	}
}

// W-04 ─ Create wallet dengan kode currency invalid → ditolak
func TestCreate_W04_InvalidCurrency(t *testing.T) {
	ownerID := createOwner(t)

	rb := createWallet(t, ownerID, "ABC")
	assertCode(t, rb, "40")
	assertContains(t, rb, "ISO 4217")
}

// W-04b ─ Create wallet dengan field kosong → ditolak
func TestCreate_W04b_MissingFields(t *testing.T) {
	// Missing sourceAccountNo
	rb1 := postJSON(t, "/core/wallets", map[string]string{
		"currency": "IDR",
	})
	assertCode(t, rb1, "40")
	assertContains(t, rb1, "wajib")

	// Missing currency
	ownerID := createOwner(t)
	rb2 := postJSON(t, "/core/wallets", map[string]string{
		"sourceAccountNo": ownerID,
	})
	assertCode(t, rb2, "40")
	assertContains(t, rb2, "wajib")
}

// W-04c ─ Create wallet dengan owner_id tidak ada → ditolak (FK violation)
func TestCreate_W04c_InvalidOwnerID(t *testing.T) {
	rb := createWallet(t, "FAKEID9999", "IDR")
	assertCode(t, rb, "50")
	assertContains(t, rb, "Owner ID")
}
