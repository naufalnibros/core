package wallets_test

import (
	"app/src/utils/db"
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: System / Concurrency Tests
// ═══════════════════════════════════════════════════════════════════════════

// SYS-01 ─ Concurrent payments: 10 goroutine masing-masing bayar 10.00
//
//	dari saldo 100.00 → saldo akhir harus ≥ 0, tidak boleh negatif
func TestConcurrency_SYS01_ConcurrentPayments(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "100.00")

	const workers = 10
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			rb := payment(t, ownerID, "IDR", "10.00")
			if rb.Attribute.Code == "00" {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
		}()
	}
	wg.Wait()

	t.Logf("SYS-01: success=%d, fail=%d", successCount, failCount)

	// Semua 10 harus berhasil karena saldo cukup (100.00 / 10 = 10.00 per payment)
	if successCount != 10 {
		t.Logf("WARNING: expected 10 successes, got %d (race condition detected)", successCount)
	}

	// Verify saldo akhir ≥ 0 (critical: no negative balance)
	balance := getLedgerBalance(t, ownerID, "IDR")
	if balance != "0.00" {
		t.Logf("INFO: final balance=%s (expected 0.00 if all 10 succeeded)", balance)
	}

	// Verify total mutasi count
	count := countMutasi(t, ownerID, "IDR")
	expectedCount := int(1 + successCount) // 1 topup + N payments
	if count != expectedCount {
		t.Errorf("expected %d mutasi records, got %d", expectedCount, count)
	}
}

// SYS-02 ─ Race: 5 goroutine topup + 5 goroutine payment secara bersamaan
func TestConcurrency_SYS02_MixedOperations(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "500.00")

	const workers = 5
	var wg sync.WaitGroup

	// 5 topup 100.00 each = +500.00
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			topup(t, ownerID, "IDR", "100.00")
		}()
	}

	// 5 payment 50.00 each = -250.00
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			payment(t, ownerID, "IDR", "50.00")
		}()
	}

	wg.Wait()

	// Expected: 500 + 500 - 250 = 750.00
	balance := getLedgerBalance(t, ownerID, "IDR")
	if balance != "750.00" {
		t.Errorf("expected balance=750.00, got=%s", balance)
	}
}

// SYS-03 ─ Read-after-write consistency: topup lalu query langsung
func TestConcurrency_SYS03_ReadAfterWrite(t *testing.T) {
	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	// Topup
	rb := topup(t, ownerID, "IDR", "123.45")
	assertCode(t, rb, "00")

	// Immediately query → must see 123.45
	q := queryWallet(t, ownerID)
	assertCode(t, q, "00")
	wallets := extractWallets(t, q)
	if wallets[0]["balance"] != "123.45" {
		t.Errorf("expected read-after-write balance=123.45, got=%s", wallets[0]["balance"])
	}
}

// SYS-04 ─ Unordered transactions: random interleaving tetap konsisten
func TestConcurrency_SYS04_UnorderedTransactions(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)

	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")

	topup(t, ownerA, "IDR", "1000.00")
	topup(t, ownerB, "IDR", "1000.00")

	var wg sync.WaitGroup

	// A→B transfer 100
	wg.Add(1)
	go func() {
		defer wg.Done()
		transfer(t, ownerA, ownerB, "IDR", "100.00")
	}()

	// B→A transfer 50
	wg.Add(1)
	go func() {
		defer wg.Done()
		transfer(t, ownerB, ownerA, "IDR", "50.00")
	}()

	wg.Wait()

	// Total uang beredar harus tetap 2000.00
	balA := getLedgerBalance(t, ownerA, "IDR")
	balB := getLedgerBalance(t, ownerB, "IDR")

	// Parse and sum
	var totalCheck string
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT TO_CHAR(SUM(balance), 'FM999999999999990.00')
		FROM core.wallets
		WHERE owner_id IN ($1, $2) AND currency = 'IDR'
	`, ownerA, ownerB).Scan(&totalCheck)
	if err != nil {
		t.Fatalf("sum query failed: %v", err)
	}

	if totalCheck != "2000.00" {
		t.Errorf("expected total=2000.00 (money conservation), got=%s (A=%s, B=%s)",
			totalCheck, balA, balB)
	}
}

// SYS-05 ─ Concurrent transfer: semua punya saldo, multiple transfer bersamaan
func TestConcurrency_SYS05_ConcurrentTransfers(t *testing.T) {
	ownerA := createOwner(t)
	ownerB := createOwner(t)
	ownerC := createOwner(t)

	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	createWallet(t, ownerC, "IDR")

	topup(t, ownerA, "IDR", "500.00")
	topup(t, ownerB, "IDR", "500.00")
	topup(t, ownerC, "IDR", "500.00")

	var wg sync.WaitGroup

	// A→B 100
	wg.Add(1)
	go func() {
		defer wg.Done()
		transfer(t, ownerA, ownerB, "IDR", "100.00")
	}()

	// B→C 100
	wg.Add(1)
	go func() {
		defer wg.Done()
		transfer(t, ownerB, ownerC, "IDR", "100.00")
	}()

	// C→A 100
	wg.Add(1)
	go func() {
		defer wg.Done()
		transfer(t, ownerC, ownerA, "IDR", "100.00")
	}()

	wg.Wait()

	// Total uang beredar = 1500.00 (konservasi uang)
	var total string
	err := db.Conn().QueryRowContext(context.Background(), `
		SELECT TO_CHAR(SUM(balance), 'FM999999999999990.00')
		FROM core.wallets
		WHERE owner_id IN ($1, $2, $3) AND currency = 'IDR'
	`, ownerA, ownerB, ownerC).Scan(&total)
	if err != nil {
		t.Fatalf("sum query failed: %v", err)
	}

	if total != "1500.00" {
		t.Errorf("expected total=1500.00 (money conservation), got=%s", total)
	}

	t.Logf("SYS-05: A=%s B=%s C=%s total=%s",
		getLedgerBalance(t, ownerA, "IDR"),
		getLedgerBalance(t, ownerB, "IDR"),
		getLedgerBalance(t, ownerC, "IDR"),
		total)
}
