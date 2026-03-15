package wallets_test

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Topup Transactions
// ═══════════════════════════════════════════════════════════════════════════

// TX-01 ─ Normal topup 50.50 → balance bertambah
func TestTopup_TX01_NormalTopup(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "50.50")
	assertCode(t, rb, "00")
	assertContains(t, rb, "Successfull")

	// Verify balance
	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "50.50" {
		t.Errorf("expected balance=50.50, got=%s", wallets[0]["balance"])
	}
}

// TX-01b ─ Topup beberapa kali → balance akumulatif
func TestTopup_TX01b_AccumulativeTopup(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "USD")

	topup(t, ownerID, "USD", "100.00")
	topup(t, ownerID, "USD", "250.75")

	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "350.75" {
		t.Errorf("expected balance=350.75, got=%s", wallets[0]["balance"])
	}
}

// TX-02 ─ Rounding: 12.345 → rounded to 12.35 (banker's rounding)
func TestTopup_TX02_Rounding(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "12.345")
	assertCode(t, rb, "00")

	// Amount di-round menjadi 12.35 (atau 12.34 tergantung implementasi)
	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	balance := wallets[0]["balance"]
	if balance != "12.35" && balance != "12.34" {
		t.Errorf("expected balance=12.35 or 12.34 (rounding), got=%s", balance)
	}
}

// TX-03 ─ Amount = 0.00 → ditolak
func TestTopup_TX03_ZeroAmount(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "0.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak boleh 0.00")
}

// TX-04 ─ Amount negatif → ditolak
func TestTopup_TX04_NegativeAmount(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "-50.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "negatif")
}

// TX-05 ─ Amount 0.001 → di bawah unit minimum (0.01) → ditolak
func TestTopup_TX05_BelowMinUnit(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "0.001")
	assertCode(t, rb, "50")
	assertContains(t, rb, "minimum")
}

// TX-07 ─ Large amount: 1,000,000,000.00 → presisi tetap terjaga
func TestTopup_TX07_LargeAmount(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "1000000000.00")
	assertCode(t, rb, "00")

	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "1000000000.00" {
		t.Errorf("expected balance=1000000000.00, got=%s", wallets[0]["balance"])
	}
}

// TX-08a ─ Topup pada wallet SUSPENDED → ditolak
func TestTopup_TX08_SuspendedWallet(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	// Suspend wallet via direct DB
	suspendWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "50.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "dibekukan")
}

// TX-Invalid-Currency ─ Topup dengan currency invalid → ditolak
func TestTopup_InvalidCurrency(t *testing.T) {
	ownerID := createOwner(t)

	rb := topup(t, ownerID, "XYZ", "100.00")
	assertCode(t, rb, "40")
	assertContains(t, rb, "ISO 4217")
}

// TX-MissingFields ─ Topup tanpa field wajib → ditolak
func TestTopup_MissingFields(t *testing.T) {
	rb := postJSON(t, "/core/wallets/topup", map[string]string{
		"sourceAccountNo": "USER1",
	})
	assertCode(t, rb, "40")
	assertContains(t, rb, "wajib")
}

// TX-WalletNotFound ─ Topup ke wallet yang belum dibuat → ditolak
func TestTopup_WalletNotFound(t *testing.T) {
	ownerID := createOwner(t)
	// Wallet belum dibuat

	rb := topup(t, ownerID, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak ditemukan")
}

// TX-InvalidAmountFormat ─ Amount bukan angka → ditolak
func TestTopup_InvalidAmountFormat(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "abc")
	assertCode(t, rb, "40")
	assertContains(t, rb, "tidak valid")
}
