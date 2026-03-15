package wallets_test

import (
	"app/src/utils/db"
	"context"
	"regexp"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Module: Owner Management — CREATE
// ═══════════════════════════════════════════════════════════════════════════

// OWN-01 ─ Owner create berhasil → code "00", owner_id format USER\d+
func TestOwner_Create_Success(t *testing.T) {
	rb := postJSON(t, "/core/owner", map[string]string{})
	assertCode(t, rb, "00")
	assertContains(t, rb, "Owner created successfully")

	result, ok := rb.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result map")
	}

	ownerID, ok := result["owner_id"].(string)
	if !ok || ownerID == "" {
		t.Fatal("expected non-empty owner_id in result")
	}

	// Verify format: USER followed by digits
	matched, _ := regexp.MatchString(`^USER\d+$`, ownerID)
	if !matched {
		t.Errorf("expected owner_id format USER\\d+, got=%q", ownerID)
	}
}

// OWN-02 ─ Owner create response structure lengkap
func TestOwner_Create_ResponseStructure(t *testing.T) {
	rb := postJSON(t, "/core/owner", map[string]string{})
	assertCode(t, rb, "00")

	// HTTP status harus 200
	if rb.HTTPStatus != 200 {
		t.Errorf("expected HTTP 200, got=%d", rb.HTTPStatus)
	}

	// txID harus non-empty
	if rb.Attribute.TxID == "" {
		t.Error("expected non-empty txID")
	}

	// result harus mengandung owner_id
	result, ok := rb.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result map")
	}
	if _, exists := result["owner_id"]; !exists {
		t.Error("expected owner_id key in result")
	}
}

// OWN-03 ─ Owner create DB failure → code "01"
// Simulasi: buat trigger BEFORE INSERT yang RAISE EXCEPTION, lalu drop setelah test
func TestOwner_Create_DBFailure(t *testing.T) {
	ctx := context.Background()

	// Buat trigger blocking function
	_, err := db.Conn().ExecContext(ctx, `
		CREATE OR REPLACE FUNCTION core.trg_block_owner_insert() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'SIMULATED_DB_ERROR: Owner insert blocked for testing';
		END;
		$$ LANGUAGE plpgsql;
	`)
	if err != nil {
		t.Fatalf("failed to create trigger function: %v", err)
	}

	// Buat trigger pada table owner
	_, err = db.Conn().ExecContext(ctx, `
		CREATE TRIGGER trg_test_block_owner_insert
		BEFORE INSERT ON core.owner
		FOR EACH ROW EXECUTE FUNCTION core.trg_block_owner_insert();
	`)
	if err != nil {
		t.Fatalf("failed to create trigger: %v", err)
	}

	// Pastikan cleanup trigger di akhir test (apapun yang terjadi)
	defer func() {
		db.Conn().ExecContext(ctx, `DROP TRIGGER IF EXISTS trg_test_block_owner_insert ON core.owner`)
		db.Conn().ExecContext(ctx, `DROP FUNCTION IF EXISTS core.trg_block_owner_insert()`)
	}()

	// Panggil owner create — harus gagal dengan code "01"
	rb := postJSON(t, "/core/owner", map[string]string{})
	assertCode(t, rb, "01")
	assertContains(t, rb, "Failed to create owner")

	// result harus nil (tidak ada owner_id)
	if rb.Result != nil {
		t.Errorf("expected nil result on failure, got: %v", rb.Result)
	}
}
