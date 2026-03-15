package wallets_test

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Transfer Transactions
// ═══════════════════════════════════════════════════════════════════════════

// TR-01 ─ Transfer standard: A → B, saldo A berkurang, saldo B bertambah
func TestTransfer_TR01_Standard(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")

	topup(t, ownerA, "IDR", "1000.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "300.00")
	assertCode(t, rb, "00")
	assertContains(t, rb, "Successful")

	// Verify saldo A = 700.00
	qA := queryWallet(t, ownerA)
	walletsA := extractWallets(t, qA)
	if walletsA[0]["balance"] != "700.00" {
		t.Errorf("A: expected balance=700.00, got=%s", walletsA[0]["balance"])
	}

	// Verify saldo B = 300.00
	qB := queryWallet(t, ownerB)
	walletsB := extractWallets(t, qB)
	if walletsB[0]["balance"] != "300.00" {
		t.Errorf("B: expected balance=300.00, got=%s", walletsB[0]["balance"])
	}
}

// TR-02 ─ Transfer ke wallet yang belum ada (tujuan tidak ditemukan)
func TestTransfer_TR02_DestinationNotFound(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	createWallet(t, ownerA, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	// ownerB belum buat wallet IDR
	rb := transfer(t, ownerA, ownerB, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak ditemukan")

	// Saldo A harus tetap 500.00 (rollback)
	qA := queryWallet(t, ownerA)
	walletsA := extractWallets(t, qA)
	if walletsA[0]["balance"] != "500.00" {
		t.Errorf("A: expected balance=500.00 (rollback), got=%s", walletsA[0]["balance"])
	}
}

// TR-02b ─ Transfer cross-currency (wallet currency berbeda) → ditolak
func TestTransfer_TR02b_CrossCurrency(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "USD") // B hanya punya USD

	topup(t, ownerA, "IDR", "1000.00")

	// Transfer IDR dari A, tapi B tidak punya wallet IDR
	rb := transfer(t, ownerA, ownerB, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak ditemukan")
}

// TR-03 ─ Transfer melebihi saldo → rollback, saldo tidak berubah
func TestTransfer_TR03_ExceedBalance(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")

	topup(t, ownerA, "IDR", "100.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "200.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak mencukupi")

	// Verify saldo A tetap 100.00
	qA := queryWallet(t, ownerA)
	walletsA := extractWallets(t, qA)
	if walletsA[0]["balance"] != "100.00" {
		t.Errorf("A: expected balance=100.00 (unchanged), got=%s", walletsA[0]["balance"])
	}

	// Verify saldo B tetap 0.00
	qB := queryWallet(t, ownerB)
	walletsB := extractWallets(t, qB)
	if walletsB[0]["balance"] != "0.00" {
		t.Errorf("B: expected balance=0.00 (unchanged), got=%s", walletsB[0]["balance"])
	}
}

// TR-SelfTransfer ─ Transfer ke diri sendiri → ditolak
func TestTransfer_SelfTransfer(t *testing.T) {
	ownerA := createOwner(t)
	createWallet(t, ownerA, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	rb := transfer(t, ownerA, ownerA, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "diri sendiri")
}

// TR-Suspended-Source ─ Transfer dari wallet SUSPENDED → ditolak
func TestTransfer_SuspendedSource(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	suspendWallet(t, ownerA, "IDR")

	rb := transfer(t, ownerA, ownerB, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "dibekukan")
}

// TR-Suspended-Dest ─ Transfer ke wallet tujuan SUSPENDED → ditolak
func TestTransfer_SuspendedDestination(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	suspendWallet(t, ownerB, "IDR")

	rb := transfer(t, ownerA, ownerB, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "dibekukan")
}

// TR-MissingFields ─ Field wajib kosong → ditolak
func TestTransfer_MissingFields(t *testing.T) {
	rb := postJSON(t, "/core/wallets/transfer", map[string]string{
		"sourceAccountNo": "USER1",
		"currency":        "IDR",
		"amount":          "100.00",
	})
	assertCode(t, rb, "40")
	assertContains(t, rb, "wajib")
}

// TR-ZeroAmount ─ Transfer amount 0 → ditolak
func TestTransfer_ZeroAmount(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "0.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak boleh 0.00")
}
