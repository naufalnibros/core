package wallets_test

import (
	"app/src/utils/db"
	"context"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Ledger Audit — Integritas Data Mutasi
// ═══════════════════════════════════════════════════════════════════════════

// LD-01 ─ SUM(topup) − SUM(payment) == balance wallet
func TestLedger_LD01_SumEqualsBalance(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	topup(t, ownerID, "IDR", "500.00")
	topup(t, ownerID, "IDR", "300.00")
	payment(t, ownerID, "IDR", "150.00")
	topup(t, ownerID, "IDR", "50.00")
	payment(t, ownerID, "IDR", "100.00")

	// Expected: 500 + 300 - 150 + 50 - 100 = 600.00
	dbBalance := getLedgerBalance(t, ownerID, "IDR")
	if dbBalance != "600.00" {
		t.Errorf("expected DB balance=600.00, got=%s", dbBalance)
	}

	// Verify via query endpoint matches DB
	q := queryWallet(t, ownerID)
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "600.00" {
		t.Errorf("expected query balance=600.00, got=%s", wallets[0]["balance"])
	}

	// Cross-check: SUM(topup) - SUM(payment) == balance
	sumTopup := getLedgerSum(t, ownerID, "IDR", "TOPUP")
	sumPayment := getLedgerSum(t, ownerID, "IDR", "PAYMENT")

	if sumTopup != "850.00" {
		t.Errorf("expected SUM(topup)=850.00, got=%s", sumTopup)
	}
	if sumPayment != "250.00" {
		t.Errorf("expected SUM(payment)=250.00, got=%s", sumPayment)
	}
}

// LD-02 ─ UPDATE pada tabel mutasi → BLOCKED oleh trigger
func TestLedger_LD02_ImmutableMutasi(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	// Coba UPDATE mutasi langsung di DB → harus error
	_, err := db.Conn().ExecContext(context.Background(), `
		UPDATE core.mutasi m
		SET amount = 999999
		FROM core.wallets w
		WHERE m.wallet_id = w.wallet_id
		  AND w.owner_id = $1 AND w.currency = 'IDR'
	`, ownerID)

	if err == nil {
		t.Fatal("expected UPDATE on mutasi to be blocked by trigger, but succeeded")
	}

	// Pastikan error berisi MUTASI:IMMUTABLE
	if msg := err.Error(); !contains(msg, "MUTASI:IMMUTABLE") {
		t.Errorf("expected error to contain MUTASI:IMMUTABLE, got: %s", msg)
	}
}

// LD-03 ─ Completeness: setelah topup + transfer, jumlah record mutasi benar
func TestLedger_LD03_CompletenessAfterTransfer(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")

	topup(t, ownerA, "IDR", "1000.00")           // +1 mutasi A
	transfer(t, ownerA, ownerB, "IDR", "300.00") // +1 TRANSFER_OUT A, +1 TRANSFER_IN B
	payment(t, ownerA, "IDR", "100.00")          // +1 mutasi A

	// Owner A harus punya 3 record mutasi (topup, transfer_out, payment)
	countA := countMutasi(t, ownerA, "IDR")
	if countA != 3 {
		t.Errorf("A: expected 3 mutasi records, got %d", countA)
	}

	// Owner B harus punya 1 record mutasi (transfer_in)
	countB := countMutasi(t, ownerB, "IDR")
	if countB != 1 {
		t.Errorf("B: expected 1 mutasi record, got %d", countB)
	}

	// Final balance check
	// A: 1000 - 300 - 100 = 600
	balA := getLedgerBalance(t, ownerA, "IDR")
	if balA != "600.00" {
		t.Errorf("A: expected balance=600.00, got=%s", balA)
	}

	// B: 0 + 300 = 300
	balB := getLedgerBalance(t, ownerB, "IDR")
	if balB != "300.00" {
		t.Errorf("B: expected balance=300.00, got=%s", balB)
	}
}

// LD-03b ─ current_balance di mutasi terakhir == balance wallet
func TestLedger_LD03b_LastMutasiBalanceMatch(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	topup(t, ownerID, "IDR", "200.00")
	payment(t, ownerID, "IDR", "50.00")
	topup(t, ownerID, "IDR", "30.00")

	// Expected: 200 - 50 + 30 = 180.00

	// current_balance dari mutasi terakhir
	var lastCurrentBalance string
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT TO_CHAR(m.current_balance, 'FM999999999999990.00')
		FROM core.mutasi m
		JOIN core.wallets w ON w.wallet_id = m.wallet_id
		WHERE w.owner_id = $1 AND w.currency = 'IDR'
		ORDER BY m.mutasi_id DESC LIMIT 1
	`, ownerID).Scan(&lastCurrentBalance)
	if err != nil {
		t.Fatalf("query last mutasi failed: %v", err)
	}

	walletBalance := getLedgerBalance(t, ownerID, "IDR")

	if lastCurrentBalance != walletBalance {
		t.Errorf("expected last mutasi current_balance=%s to match wallet balance=%s",
			lastCurrentBalance, walletBalance)
	}
}

// ─── helper ─────────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSub(s, substr))
}

func containsSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
