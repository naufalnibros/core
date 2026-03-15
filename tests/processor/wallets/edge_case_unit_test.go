package wallets_test

import (
	"app/src/utils/db"
	"context"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Edge Cases & Boundary Conditions
// ═══════════════════════════════════════════════════════════════════════════

// ─── Currency Edge Cases ─────────────────────────────────────────────────

// EC-01 ─ Currency lowercase "idr" → ditolak (ISO 4217 map case-sensitive)
func TestEdge_CurrencyLowercase(t *testing.T) {
	ownerID := createOwner(t)

	rb := createWallet(t, ownerID, "idr")
	assertCode(t, rb, "40")
	assertContains(t, rb, "ISO 4217")
}

// ─── Amount Edge Cases ──────────────────────────────────────────────────

// EC-02 ─ Amount tanpa desimal "100" → harus berhasil, balance "100.00"
func TestEdge_AmountWithoutDecimals(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "100")
	assertCode(t, rb, "00")

	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "100.00" {
		t.Errorf("expected balance=100.00, got=%s", wallets[0]["balance"])
	}
}

// EC-03 ─ Amount exact minimum unit "0.01" (boundary) → harus berhasil
func TestEdge_AmountExactMinUnit(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "0.01")
	assertCode(t, rb, "00")

	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "0.01" {
		t.Errorf("expected balance=0.01, got=%s", wallets[0]["balance"])
	}
}

// EC-04 ─ Amount dengan comma separator "1,000.00" → invalid format
func TestEdge_AmountWithComma(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "1,000.00")
	assertCode(t, rb, "40")
	assertContains(t, rb, "tidak valid")
}

// ─── Unicode Field Edge Case ────────────────────────────────────────────

// EC-05 ─ Verifikasi bahwa field beneﬁciaryAccountNo menggunakan Unicode ﬁ (U+FB01)
//
//	Client yang kirim ASCII "beneficiaryAccountNo" (fi biasa) → field kosong → validation error
func TestEdge_UnicodeFieldBeneficiary(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	// Kirim dengan ASCII "fi" biasa (bukan Unicode ﬁ U+FB01)
	// Field akan kosong karena struct tag menggunakan "beneﬁciaryAccountNo" (U+FB01)
	rb := postJSON(t, "/core/wallets/transfer", map[string]string{
		"sourceAccountNo":      ownerA,
		"beneficiaryAccountNo": ownerB, // ASCII "fi" — NOT matching struct tag
		"currency":             "IDR",
		"amount":               "100.00",
	})

	// Harus ditolak karena beneﬁciaryAccountNo (Unicode) akan kosong
	assertCode(t, rb, "40")
	assertContains(t, rb, "wajib")
}

// ─── SQL Injection Safety ───────────────────────────────────────────────

// EC-06 ─ SQL injection di sourceAccountNo → parameterized query aman
func TestEdge_SQLInjectionSourceAccountNo(t *testing.T) {
	// Attempt SQL injection — must not crash, must return proper error
	rb := createWallet(t, "'; DROP TABLE wallets; --", "IDR")

	// Harus error (FK violation atau format error), BUKAN crash
	if rb.Attribute.Code == "" {
		t.Fatal("expected a response code, got empty (possible crash)")
	}

	// Pastikan code bukan "00" (tidak boleh berhasil)
	if rb.Attribute.Code == "00" {
		t.Fatal("SQL injection should NOT succeed")
	}
}

// ─── Reactivation Lifecycle ─────────────────────────────────────────────

// EC-07 ─ Suspend → fail topup → activate → topup berhasil → balance benar
func TestEdge_ReactivationLifecycle(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	// Step 1: Suspend
	suspendWallet(t, ownerID, "IDR")

	// Step 2: Topup harus gagal
	rb := topup(t, ownerID, "IDR", "50.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "dibekukan")

	// Step 3: Activate kembali
	activateWallet(t, ownerID, "IDR")

	// Step 4: Topup harus berhasil
	rb2 := topup(t, ownerID, "IDR", "50.00")
	assertCode(t, rb2, "00")

	// Step 5: Verify balance = 100 + 50 = 150
	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "150.00" {
		t.Errorf("expected balance=150.00, got=%s", wallets[0]["balance"])
	}
	if wallets[0]["status"] != "ACTIVE" {
		t.Errorf("expected status=ACTIVE, got=%s", wallets[0]["status"])
	}
}

// ─── Payment Parity Tests (missing scenarios from topup) ────────────────

// EC-08 ─ Payment rounding: 12.345 → di-round ke 12.35
func TestEdge_PaymentRounding(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	rb := payment(t, ownerID, "IDR", "12.345")
	assertCode(t, rb, "00")

	// 100.00 - 12.35 (rounded) = 87.65
	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "87.65" {
		t.Errorf("expected balance=87.65 (after rounding), got=%s", wallets[0]["balance"])
	}
}

// EC-09 ─ Payment below minimum unit 0.001 → ditolak
func TestEdge_PaymentBelowMinUnit(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	rb := payment(t, ownerID, "IDR", "0.001")
	assertCode(t, rb, "50")
	assertContains(t, rb, "minimum")
}

// EC-10 ─ Payment invalid amount format "abc" → ditolak
func TestEdge_PaymentInvalidAmountFormat(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := payment(t, ownerID, "IDR", "abc")
	assertCode(t, rb, "40")
	assertContains(t, rb, "tidak valid")
}

// EC-11 ─ Payment large amount (999999999 dari saldo 1B) → presisi terjaga
func TestEdge_PaymentLargeAmount(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "1000000000.00")

	rb := payment(t, ownerID, "IDR", "999999999.00")
	assertCode(t, rb, "00")

	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "1.00" {
		t.Errorf("expected balance=1.00, got=%s", wallets[0]["balance"])
	}
}

// ─── Transfer Parity Tests (missing scenarios) ─────────────────────────

// EC-12 ─ Transfer negative amount → ditolak
func TestEdge_TransferNegativeAmount(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "-100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "negatif")
}

// EC-13 ─ Transfer invalid amount format "abc" → ditolak
func TestEdge_TransferInvalidAmountFormat(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "abc")
	assertCode(t, rb, "40")
	assertContains(t, rb, "tidak valid")
}

// EC-14 ─ Transfer invalid currency → ditolak
func TestEdge_TransferInvalidCurrency(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	rb := transfer(t, ownerA, ownerB, "XXX", "100.00")
	assertCode(t, rb, "40")
	assertContains(t, rb, "ISO 4217")
}

// EC-15 ─ Transfer below minimum unit 0.001 → ditolak
func TestEdge_TransferBelowMinUnit(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "0.001")
	assertCode(t, rb, "50")
	assertContains(t, rb, "minimum")
}

// EC-16 ─ Transfer rounding: 12.345 → di-round ke 12.35
func TestEdge_TransferRounding(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "1000.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "12.345")
	assertCode(t, rb, "00")

	// A: 1000 - 12.35 = 987.65
	qA := queryWallet(t, ownerA)
	walletsA := extractWallets(t, qA)
	if walletsA[0]["balance"] != "987.65" {
		t.Errorf("A: expected balance=987.65, got=%s", walletsA[0]["balance"])
	}

	// B: 0 + 12.35 = 12.35
	qB := queryWallet(t, ownerB)
	walletsB := extractWallets(t, qB)
	if walletsB[0]["balance"] != "12.35" {
		t.Errorf("B: expected balance=12.35, got=%s", walletsB[0]["balance"])
	}
}

// EC-17 ─ Transfer seluruh saldo → sender balance = 0.00
func TestEdge_TransferExactBalance(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "500.00")
	assertCode(t, rb, "00")

	// A: 0.00
	qA := queryWallet(t, ownerA)
	walletsA := extractWallets(t, qA)
	if walletsA[0]["balance"] != "0.00" {
		t.Errorf("A: expected balance=0.00, got=%s", walletsA[0]["balance"])
	}

	// B: 500.00
	qB := queryWallet(t, ownerB)
	walletsB := extractWallets(t, qB)
	if walletsB[0]["balance"] != "500.00" {
		t.Errorf("B: expected balance=500.00, got=%s", walletsB[0]["balance"])
	}
}

// EC-18 ─ Transfer: both source AND destination SUSPENDED
func TestEdge_TransferBothSuspended(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "500.00")

	suspendWallet(t, ownerA, "IDR")
	suspendWallet(t, ownerB, "IDR")

	rb := transfer(t, ownerA, ownerB, "IDR", "100.00")
	assertCode(t, rb, "50")
	assertContains(t, rb, "dibekukan")
}

// ─── Mutasi Audit Trail Tests ───────────────────────────────────────────

// EC-19 ─ Verify mutasi record terbuat setelah topup
func TestEdge_MutasiAudit_Topup(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	rb := topup(t, ownerID, "IDR", "250.50")
	assertCode(t, rb, "00")

	// Verify mutasi record
	var tipe, amount, currentBalance, keterangan string
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT m.tipe, 
			   TO_CHAR(m.amount, 'FM999999999999990.00') AS amount,
			   TO_CHAR(m.current_balance, 'FM999999999999990.00') AS current_balance,
			   m.keterangan
		FROM core.mutasi m
		JOIN core.wallets w ON w.wallet_id = m.wallet_id
		WHERE w.owner_id = $1 AND w.currency = 'IDR'
		ORDER BY m.mutasi_id DESC LIMIT 1
	`, ownerID).Scan(&tipe, &amount, &currentBalance, &keterangan)
	if err != nil {
		t.Fatalf("query mutasi failed: %v", err)
	}

	if tipe != "TOPUP" {
		t.Errorf("expected tipe=TOPUP, got=%s", tipe)
	}
	if amount != "250.50" {
		t.Errorf("expected amount=250.50, got=%s", amount)
	}
	if currentBalance != "250.50" {
		t.Errorf("expected current_balance=250.50, got=%s", currentBalance)
	}
	if keterangan == "" {
		t.Error("expected non-empty keterangan")
	}
}

// EC-20 ─ Verify mutasi record terbuat setelah payment
func TestEdge_MutasiAudit_Payment(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "500.00")

	rb := payment(t, ownerID, "IDR", "150.00")
	assertCode(t, rb, "00")

	// Verify mutasi record for payment
	var tipe, amount, currentBalance string
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT m.tipe, 
			   TO_CHAR(m.amount, 'FM999999999999990.00') AS amount,
			   TO_CHAR(m.current_balance, 'FM999999999999990.00') AS current_balance
		FROM core.mutasi m
		JOIN core.wallets w ON w.wallet_id = m.wallet_id
		WHERE w.owner_id = $1 AND w.currency = 'IDR' AND m.tipe = 'PAYMENT'
		ORDER BY m.mutasi_id DESC LIMIT 1
	`, ownerID).Scan(&tipe, &amount, &currentBalance)
	if err != nil {
		t.Fatalf("query mutasi failed: %v", err)
	}

	if tipe != "PAYMENT" {
		t.Errorf("expected tipe=PAYMENT, got=%s", tipe)
	}
	if amount != "150.00" {
		t.Errorf("expected amount=150.00, got=%s", amount)
	}
	// 500 - 150 = 350
	if currentBalance != "350.00" {
		t.Errorf("expected current_balance=350.00, got=%s", currentBalance)
	}
}

// EC-21 ─ Verify 2 mutasi records (TRANSFER_OUT + TRANSFER_IN) setelah transfer
func TestEdge_MutasiAudit_Transfer(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "1000.00")

	rb := transfer(t, ownerA, ownerB, "IDR", "300.00")
	assertCode(t, rb, "00")

	// Verify TRANSFER_OUT on sender
	var tipeOut, amountOut, balOut, ketOut string
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT m.tipe, 
			   TO_CHAR(m.amount, 'FM999999999999990.00'),
			   TO_CHAR(m.current_balance, 'FM999999999999990.00'),
			   m.keterangan
		FROM core.mutasi m
		JOIN core.wallets w ON w.wallet_id = m.wallet_id
		WHERE w.owner_id = $1 AND w.currency = 'IDR' AND m.tipe = 'TRANSFER_OUT'
		ORDER BY m.mutasi_id DESC LIMIT 1
	`, ownerA).Scan(&tipeOut, &amountOut, &balOut, &ketOut)
	if err != nil {
		t.Fatalf("query TRANSFER_OUT failed: %v", err)
	}

	if tipeOut != "TRANSFER_OUT" {
		t.Errorf("expected tipe=TRANSFER_OUT, got=%s", tipeOut)
	}
	if amountOut != "300.00" {
		t.Errorf("expected TRANSFER_OUT amount=300.00, got=%s", amountOut)
	}
	if balOut != "700.00" {
		t.Errorf("expected TRANSFER_OUT current_balance=700.00, got=%s", balOut)
	}
	if ketOut == "" {
		t.Error("expected non-empty keterangan for TRANSFER_OUT")
	}

	// Verify TRANSFER_IN on receiver
	var tipeIn, amountIn, balIn, ketIn string
	err = db.Conn().QueryRowContext(context.Background(), `
		SELECT m.tipe, 
			   TO_CHAR(m.amount, 'FM999999999999990.00'),
			   TO_CHAR(m.current_balance, 'FM999999999999990.00'),
			   m.keterangan
		FROM core.mutasi m
		JOIN core.wallets w ON w.wallet_id = m.wallet_id
		WHERE w.owner_id = $1 AND w.currency = 'IDR' AND m.tipe = 'TRANSFER_IN'
		ORDER BY m.mutasi_id DESC LIMIT 1
	`, ownerB).Scan(&tipeIn, &amountIn, &balIn, &ketIn)
	if err != nil {
		t.Fatalf("query TRANSFER_IN failed: %v", err)
	}

	if tipeIn != "TRANSFER_IN" {
		t.Errorf("expected tipe=TRANSFER_IN, got=%s", tipeIn)
	}
	if amountIn != "300.00" {
		t.Errorf("expected TRANSFER_IN amount=300.00, got=%s", amountIn)
	}
	if balIn != "300.00" {
		t.Errorf("expected TRANSFER_IN current_balance=300.00, got=%s", balIn)
	}
	if ketIn == "" {
		t.Error("expected non-empty keterangan for TRANSFER_IN")
	}
}
