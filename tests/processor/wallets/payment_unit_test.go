package wallets_test

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Payment Transactions
// ═══════════════════════════════════════════════════════════════════════════

// TX-06 ─ Payment melebihi saldo → ditolak (CHECK balance >= 0)
func TestPayment_TX06_ExceedBalance(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	rb := payment(t, ownerID, "IDR", "150.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak mencukupi")
}

// TX-06b ─ Payment normal (saldo cukup) → berhasil
func TestPayment_TX06b_NormalPayment(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "500.00")

	rb := payment(t, ownerID, "IDR", "200.00")
	assertCode(t, rb, "00")
	assertContains(t, rb, "Successful")

	// Verify balance = 300.00
	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "300.00" {
		t.Errorf("expected balance=300.00, got=%s", wallets[0]["balance"])
	}
}

// TX-06c ─ Payment exact balance → saldo jadi 0.00
func TestPayment_TX06c_ExactBalance(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	rb := payment(t, ownerID, "IDR", "100.00")
	assertCode(t, rb, "00")

	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "0.00" {
		t.Errorf("expected balance=0.00, got=%s", wallets[0]["balance"])
	}
}

// TX-08b ─ Payment pada wallet SUSPENDED → ditolak
func TestPayment_TX08_SuspendedWallet(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "500.00")

	suspendWallet(t, ownerID, "IDR")

	rb := payment(t, ownerID, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "dibekukan")
}

// Payment-ZeroAmount ─ Payment amount 0 → ditolak
func TestPayment_ZeroAmount(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	rb := payment(t, ownerID, "IDR", "0.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak boleh 0.00")
}

// Payment-NegativeAmount ─ Payment amount negatif → ditolak
func TestPayment_NegativeAmount(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	rb := payment(t, ownerID, "IDR", "-10.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "negatif")
}

// Payment-MissingFields ─ Field wajib kosong → ditolak
func TestPayment_MissingFields(t *testing.T) {
	rb := postJSON(t, "/core/wallets/payment", map[string]string{
		"sourceAccountNo": "USER1",
		"currency":        "IDR",
	})
	assertCode(t, rb, "40")
	assertContains(t, rb, "wajib")
}

// Payment-InvalidCurrency ─ Currency invalid → ditolak
func TestPayment_InvalidCurrency(t *testing.T) {
	ownerID := createOwner(t)

	rb := payment(t, ownerID, "ZZZ", "100.00")
	assertCode(t, rb, "40")
	assertContains(t, rb, "ISO 4217")
}

// Payment-WalletNotFound ─ Wallet belum dibuat → ditolak
func TestPayment_WalletNotFound(t *testing.T) {
	ownerID := createOwner(t)

	rb := payment(t, ownerID, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "tidak ditemukan")
}
