package wallets_test

import (
	"app/src/utils/cache"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/shopspring/decimal"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Idempotency — Duplicate Transaction Detection
// Requires: Memcached running at localhost:11511 (via Docker Compose)
// ═══════════════════════════════════════════════════════════════════════════

// connectMemcached sets cache.MC to a real Memcached client for idempotency tests.
// Returns a cleanup function that restores cache.MC to nil.
func connectMemcached(t *testing.T) func() {
	t.Helper()

	original := cache.MC
	cache.MC = memcache.New("localhost:11511")

	if err := cache.MC.Ping(); err != nil {
		cache.MC = original
		t.Skipf("Memcached not available at localhost:11511, skipping: %v", err)
	}

	return func() {
		cache.MC = original
	}
}

// flushKey removes a specific idempotency lock key from Memcached.
func flushKey(t *testing.T, key string) {
	t.Helper()
	cache.MC.Delete(cache.IdempotencyPrefix + key)
}

// dupKey builds the same DUPLICATE_KEY that the handler generates.
// Important: handler uses decimal.NewFromString(amount).String() which strips trailing zeros.
// e.g. "100.00" → "100", "12.35" → "12.35", "0.50" → "0.5"
func dupKey(operation, ownerID, amount, currency string, beneficiary ...string) string {
	// Replicate handler logic: decimal.NewFromString strips trailing zeros
	d, _ := decimal.NewFromString(amount)
	key := operation + ":" + ownerID + ":" + d.String() + ":" + currency
	if len(beneficiary) > 0 {
		key += ":" + beneficiary[0]
	}
	return key
}

// ID-01 ─ Duplicate topup → second request blocked with code "83"
func TestIdempotency_DuplicateTopup(t *testing.T) {
	cleanup := connectMemcached(t)
	defer cleanup()

	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	// Build the same duplicate key the handler would generate
	duplicateKey := dupKey("TOPUP", ownerID, "100.00", "IDR")

	// Flush any leftover lock from previous runs
	flushKey(t, duplicateKey)

	// First topup → should succeed
	rb1 := topup(t, ownerID, "IDR", "100.00")
	assertCode(t, rb1, "00")

	// Second identical topup → should be blocked as duplicate
	rb2 := topup(t, ownerID, "IDR", "100.00")

	if rb2.HTTPStatus != 425 {
		t.Errorf("expected HTTP 425 (Too Early), got=%d", rb2.HTTPStatus)
	}
	assertCode(t, rb2, "83")
	assertContains(t, rb2, "Duplicate")

	// Balance should remain 100.00 (not 200.00)
	rb3 := queryWallet(t, ownerID)
	assertCode(t, rb3, "00")
	ws := extractWallets(t, rb3)
	if ws[0]["balance"] != "100.00" {
		t.Errorf("expected balance 100.00 after duplicate block, got=%s", ws[0]["balance"])
	}

	// Cleanup lock
	flushKey(t, duplicateKey)
}

// ID-02 ─ Duplicate payment → second request blocked with code "83"
func TestIdempotency_DuplicatePayment(t *testing.T) {
	cleanup := connectMemcached(t)
	defer cleanup()

	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")
	topup(t, ownerID, "IDR", "500.00")

	duplicateKey := dupKey("PAYMENT", ownerID, "50.00", "IDR")
	flushKey(t, duplicateKey)

	// First payment → should succeed
	rb1 := payment(t, ownerID, "IDR", "50.00")
	assertCode(t, rb1, "00")

	// Second identical payment → should be blocked
	rb2 := payment(t, ownerID, "IDR", "50.00")
	if rb2.HTTPStatus != 425 {
		t.Errorf("expected HTTP 425 (Too Early), got=%d", rb2.HTTPStatus)
	}
	assertCode(t, rb2, "83")
	assertContains(t, rb2, "Duplicate")

	// Balance should be 450.00 (only one payment processed)
	rb3 := queryWallet(t, ownerID)
	assertCode(t, rb3, "00")
	ws := extractWallets(t, rb3)
	if ws[0]["balance"] != "450.00" {
		t.Errorf("expected balance 450.00, got=%s", ws[0]["balance"])
	}

	flushKey(t, duplicateKey)
}

// ID-03 ─ Duplicate transfer → second request blocked with code "83"
func TestIdempotency_DuplicateTransfer(t *testing.T) {
	cleanup := connectMemcached(t)
	defer cleanup()

	ownerA := createOwner(t)
	ownerB := createOwner(t)
	createWallet(t, ownerA, "IDR")
	createWallet(t, ownerB, "IDR")
	topup(t, ownerA, "IDR", "1000.00")

	duplicateKey := dupKey("TRANSFER", ownerA, "200.00", "IDR", ownerB)
	flushKey(t, duplicateKey)

	// First transfer → should succeed
	rb1 := transfer(t, ownerA, ownerB, "IDR", "200.00")
	assertCode(t, rb1, "00")

	// Second identical transfer → should be blocked
	rb2 := transfer(t, ownerA, ownerB, "IDR", "200.00")
	if rb2.HTTPStatus != 425 {
		t.Errorf("expected HTTP 425 (Too Early), got=%d", rb2.HTTPStatus)
	}
	assertCode(t, rb2, "83")
	assertContains(t, rb2, "Duplicate")

	// Balance A should be 800.00, B should be 200.00 (only one transfer)
	rbA := queryWallet(t, ownerA)
	assertCode(t, rbA, "00")
	wsA := extractWallets(t, rbA)
	if wsA[0]["balance"] != "800.00" {
		t.Errorf("expected A balance 800.00, got=%s", wsA[0]["balance"])
	}

	rbB := queryWallet(t, ownerB)
	assertCode(t, rbB, "00")
	wsB := extractWallets(t, rbB)
	if wsB[0]["balance"] != "200.00" {
		t.Errorf("expected B balance 200.00, got=%s", wsB[0]["balance"])
	}

	flushKey(t, duplicateKey)
}

// ID-04 ─ Different amounts are NOT duplicates (different DUPLICATE_KEY)
func TestIdempotency_DifferentAmountsNotDuplicate(t *testing.T) {
	cleanup := connectMemcached(t)
	defer cleanup()

	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	key1 := dupKey("TOPUP", ownerID, "100.00", "IDR")
	key2 := dupKey("TOPUP", ownerID, "200.00", "IDR")
	flushKey(t, key1)
	flushKey(t, key2)

	// Topup 100 → succeed
	rb1 := topup(t, ownerID, "IDR", "100.00")
	assertCode(t, rb1, "00")

	// Topup 200 (different amount) → should also succeed (different key)
	rb2 := topup(t, ownerID, "IDR", "200.00")
	assertCode(t, rb2, "00")

	// Balance should be 300.00 (both processed)
	rb3 := queryWallet(t, ownerID)
	assertCode(t, rb3, "00")
	ws := extractWallets(t, rb3)
	if ws[0]["balance"] != "300.00" {
		t.Errorf("expected balance 300.00, got=%s", ws[0]["balance"])
	}

	flushKey(t, key1)
	flushKey(t, key2)
}

// ID-05 ─ Lock expires after TTL → retry should succeed
func TestIdempotency_LockExpiresAfterTTL(t *testing.T) {
	cleanup := connectMemcached(t)
	defer cleanup()

	ownerID := createOwner(t)
	createWallet(t, ownerID, "IDR")

	duplicateKey := dupKey("TOPUP", ownerID, "75.00", "IDR")
	flushKey(t, duplicateKey)

	// Manually set a short-lived lock (1 second TTL)
	cache.MC.Add(&memcache.Item{
		Key:        cache.IdempotencyPrefix + duplicateKey,
		Value:      []byte("1"),
		Expiration: 1, // 1 second
	})

	// Immediately → should be blocked
	rb1 := topup(t, ownerID, "IDR", "75.00")
	if rb1.HTTPStatus != 425 {
		t.Errorf("expected HTTP 425 while lock active, got=%d", rb1.HTTPStatus)
	}
	assertCode(t, rb1, "83")

	// Wait for lock to expire
	time.Sleep(2 * time.Second)

	// After expiry → should succeed
	rb2 := topup(t, ownerID, "IDR", "75.00")
	assertCode(t, rb2, "00")

	// Balance should be 75.00
	rb3 := queryWallet(t, ownerID)
	ws := extractWallets(t, rb3)
	if ws[0]["balance"] != "75.00" {
		t.Errorf("expected balance 75.00, got=%s", ws[0]["balance"])
	}

	flushKey(t, duplicateKey)
}
