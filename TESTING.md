# 📋 Dokumentasi Unit Test — SERVICE CORE

> **Proyek:** `app` (Go/Fiber Service Core)  
> **Lokasi Test:** `tests/processor/wallets/`  
> **Total Test:** 82  
> **Terakhir Diperbarui:** 15 Maret 2026 21:00

---

## 🚀 Cara Menjalankan Unit Test

### Prasyarat

1. **Docker & Docker Compose** terinstall
2. **Go 1.24+** terinstall
3. **File `.env`** sudah terkonfigurasi (salin dari `.env.example` jika belum ada)

### Langkah 1 — Jalankan Semua Service (Wajib)

Sebelum menjalankan test, **wajib** menjalankan seluruh service melalui Docker Compose:

```bash
docker compose up -d
```

Perintah ini akan menjalankan:

| Service | Image | Container | Port Host | Keterangan |
|---|---|---|---|---|
| **PostgreSQL** | `postgres:16-alpine` | `core-postgres` | `9832` | Database utama, schema `core` |
| **Memcached** | `memcached:1.6-alpine` | `core-memcached` | `11511` | Cache untuk idempotency / deteksi duplikat |
| **App** | (Dockerfile) | `core-app` | `11082` | Aplikasi (tidak diperlukan untuk test, tapi ikut jalan) |

Pastikan semua service sudah **healthy** sebelum menjalankan test:

```bash
docker compose ps
```

Output yang diharapkan (kolom STATUS):

```
core-postgres    ... healthy
core-memcached   ... healthy
core-app         ... running
```

### Langkah 2 — Jalankan Test

```bash
# Dari root project — jalankan semua test
go test ./tests/processor/wallets/ -v -count=1 -timeout=120s
```

> **Catatan:** Tidak perlu set `DB_TEST` secara manual. Variabel sudah dikonfigurasi di file `.env` dan otomatis di-load oleh `env.Init()` saat test dijalankan.

### Menjalankan Test Tertentu

```bash
# Satu test spesifik
go test ./tests/processor/wallets/ -v -count=1 -run TestCreate_W01_NewWallet

# Berdasarkan pola nama (semua test Transfer)
go test ./tests/processor/wallets/ -v -count=1 -run "TestTransfer_"

# Semua test dalam satu kategori
go test ./tests/processor/wallets/ -v -count=1 -run "TestLedger_"
go test ./tests/processor/wallets/ -v -count=1 -run "TestIdempotency_"
go test ./tests/processor/wallets/ -v -count=1 -run "TestEdge_"
```

### Konfigurasi Environment (`.env`)

| Variabel | Contoh Nilai | Keterangan |
|---|---|---|
| `DB_TEST` | `postgres://coredev:coredev@localhost:9832/core?sslmode=disable&search_path=core` | DSN koneksi ke database test. Otomatis dari `.env` |
| `DEPLOYMENT_MODE` | `dev` | Jika `dev`, field `errormsg` di response akan diisi |
| `MEMCACHED_HOST` | `localhost:11511` | Host Memcached. Digunakan oleh test idempotency |

### Catatan Penting

- Flag `-count=1` → memastikan test **tidak menggunakan cache** hasil sebelumnya
- Flag `-timeout=120s` → mencegah timeout pada test concurrency
- Semua test bersifat **independen** — setiap test membuat owner & wallet baru
- Test idempotency otomatis **di-skip** (`t.Skipf`) jika Memcached tidak tersedia

---

## 📁 Struktur File Test

```
tests/
├── setup.go                            # Factory untuk test app (NewTestApp, NewTestAppWithDB)
└── processor/wallets/
    ├── helper_test.go                   # Fungsi & helper test
    ├── create_unit_test.go              # Test pembuatan wallet
    ├── topup_unit_test.go               # Test transaksi topup
    ├── payment_unit_test.go             # Test transaksi payment
    ├── transfer_unit_test.go            # Test transaksi transfer
    ├── query_unit_test.go               # Test query status wallet
    ├── ledger_unit_test.go              # Test audit & integritas ledger
    ├── concurrency_unit_test.go         # Test concurrency & race condition
    ├── owner_unit_test.go               # Test endpoint owner
    ├── routing_unit_test.go             # Test routing & middleware
    ├── edge_case_unit_test.go           # Test edge case & boundary
    └── idempotency_unit_test.go         # Test idempotency / deteksi duplikat
```

---

## 🔧 Fungsi Helper — `helper_test.go`

Fungsi-fungsi di bawah ini digunakan bersama oleh seluruh file test.

| Fungsi | Kegunaan | Merujuk ke |
|---|---|---|
| `getApp()` | Singleton Fiber app + koneksi DB untuk test | `tests.NewTestAppWithDB()` |
| `jsonBody(m)` | Konversi `map[string]string` → `*bytes.Buffer` JSON | — |
| `postJSON(t, url, body)` | Kirim HTTP POST + unmarshal response | — |
| `createOwner(t)` | Buat owner baru via endpoint, kembalikan `owner_id` | `POST /core/owner` → `owner.Create` |
| `createWallet(t, ownerID, currency)` | Buat wallet baru | `POST /core/wallets` → `wallets.Create` |
| `topup(t, ownerID, currency, amount)` | Topup saldo wallet | `POST /core/wallets/topup` → `wallets.Topup` |
| `payment(t, ownerID, currency, amount)` | Pembayaran dari wallet | `POST /core/wallets/payment` → `wallets.Payment` |
| `transfer(t, from, to, currency, amount)` | Transfer antar wallet | `POST /core/wallets/transfer` → `wallets.Transfer` |
| `queryWallet(t, ownerID)` | Query semua wallet milik owner | `POST /core/wallets/query` → `wallets.Query` |
| `assertCode(t, rb, expected)` | Assert response code sesuai ekspektasi | — |
| `assertContains(t, rb, substr)` | Assert message mengandung substring | — |
| `extractWallets(t, rb)` | Ekstrak array wallet dari response | — |
| `suspendWallet(t, ownerID, currency)` | Bekukan wallet via DB langsung | `UPDATE core.wallets SET status = 'SUSPENDED'` |
| `activateWallet(t, ownerID, currency)` | Aktifkan wallet via DB langsung | `UPDATE core.wallets SET status = 'ACTIVE'` |
| `getLedgerSum(t, ownerID, currency, tipe)` | Total amount mutasi per tipe | `SELECT SUM(m.amount) FROM core.mutasi` |
| `getLedgerBalance(t, ownerID, currency)` | Saldo wallet dari DB | `SELECT balance FROM core.wallets` |
| `countMutasi(t, ownerID, currency)` | Hitung jumlah record mutasi | `SELECT COUNT(*) FROM core.mutasi` |

Fungsi helper khusus idempotency (di `idempotency_unit_test.go`):

| Fungsi | Kegunaan | Merujuk ke |
|---|---|---|
| `connectMemcached(t)` | Koneksi ke Memcached, kembalikan fungsi cleanup | `cache.MC = memcache.New("localhost:11511")` |
| `flushKey(t, key)` | Hapus lock key dari Memcached | `cache.MC.Delete(...)` |
| `dupKey(op, owner, amount, currency, ...)` | Bangun DUPLICATE_KEY sesuai format handler | `decimal.NewFromString(amount).String()` |

---

## ✅ Daftar Seluruh Skenario Test (82 Test)

### 1. Pembuatan Wallet — `create_unit_test.go` (6 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| W-01 | `TestCreate_W01_NewWallet` | Buat wallet baru IDR | code `"00"`, saldo `"0.00"`, status `"ACTIVE"` | `wallets.Create` → `exec.Wallet().Create()` |
| W-02 | `TestCreate_W02_DuplicateCurrency` | Buat wallet duplikat (IDR 2x) | code `"50"`, pesan mengandung `"duplicate"` | `exec.ErrWallet` → constraint `wallet_unique_currency_per_owner` |
| W-03 | `TestCreate_W03_MultiCurrency` | Buat IDR + USD + EUR untuk satu owner | code `"00"` 3x, query mengembalikan 3 wallet | `wallets.Create` (3 kali) |
| W-04 | `TestCreate_W04_InvalidCurrency` | Currency `"ABC"` (bukan ISO 4217) | code `"40"`, pesan mengandung `"ISO 4217"` | `currency.Cek()` |
| W-04b | `TestCreate_W04b_MissingFields` | Field `sourceAccountNo` / `currency` kosong | code `"40"`, pesan mengandung `"wajib"` | Validasi field di `wallets.Create` |
| W-04c | `TestCreate_W04c_InvalidOwnerID` | Owner ID `"FAKEID9999"` tidak terdaftar | code `"50"`, pesan mengandung `"Owner ID"` | `exec.ErrWallet` → constraint `wallets_owner_id_fkey` |

### 2. Topup — `topup_unit_test.go` (12 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| TX-01 | `TestTopup_TX01_NormalTopup` | Topup 50.50 IDR | code `"00"`, saldo `"50.50"` | `exec.Wallet().Topup()` |
| TX-01b | `TestTopup_TX01b_AccumulativeTopup` | Topup 100 + 250.75 USD | saldo `"350.75"` | `exec.Wallet().Topup()` (2 kali) |
| TX-02 | `TestTopup_TX02_Rounding` | Topup `"12.345"` | saldo dibulatkan `"12.35"` | `validasiAmount()` → `Round(2)` |
| TX-03 | `TestTopup_TX03_ZeroAmount` | Amount `"0.00"` | code `"50"`, pesan `"tidak boleh 0.00"` | `validasiAmount()` |
| TX-04 | `TestTopup_TX04_NegativeAmount` | Amount `"-50.00"` | code `"50"`, pesan `"negatif"` | `validasiAmount()` |
| TX-05 | `TestTopup_TX05_BelowMinUnit` | Amount `"0.001"` | code `"50"`, pesan `"minimum"` | `validasiAmount()` → `minUnit` |
| TX-07 | `TestTopup_TX07_LargeAmount` | Amount `"1000000000.00"` (1 Miliar) | saldo `"1000000000.00"` | Presisi `NUMERIC(19,4)` |
| TX-08 | `TestTopup_TX08_SuspendedWallet` | Topup pada wallet yang dibekukan | code `"50"`, pesan `"dibekukan"` | Trigger `trg_check_wallet_status_mutasi` |
| — | `TestTopup_InvalidCurrency` | Currency `"XYZ"` tidak valid | code `"40"` | `currency.Cek()` |
| — | `TestTopup_MissingFields` | Field wajib tidak diisi | code `"40"` | Validasi field |
| — | `TestTopup_WalletNotFound` | Wallet belum dibuat | code `"50"` | `sql.ErrNoRows` |
| — | `TestTopup_InvalidAmountFormat` | Amount `"abc"` (bukan angka) | code `"40"` | `req.ParseAmount()` |

### 3. Payment — `payment_unit_test.go` (9 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| TX-06 | `TestPayment_TX06_ExceedBalance` | Pembayaran melebihi saldo | code `"50"`, pesan `"tidak mencukupi"` | Constraint `wallet_cek_balance` |
| TX-06b | `TestPayment_TX06b_NormalPayment` | Bayar 200 dari saldo 500 | saldo `"300.00"` | `exec.Wallet().Payment()` |
| TX-06c | `TestPayment_TX06c_ExactBalance` | Bayar = saldo (100) | saldo `"0.00"` | `exec.Wallet().Payment()` |
| TX-08 | `TestPayment_TX08_SuspendedWallet` | Bayar dari wallet yang dibekukan | code `"50"`, pesan `"dibekukan"` | Trigger `trg_check_wallet_status_mutasi` |
| — | `TestPayment_ZeroAmount` | Amount `"0.00"` | code `"50"` | `validasiAmount()` |
| — | `TestPayment_NegativeAmount` | Amount `"-10.00"` | code `"50"` | `validasiAmount()` |
| — | `TestPayment_MissingFields` | Field `amount` kosong | code `"40"` | Validasi field |
| — | `TestPayment_InvalidCurrency` | Currency `"ZZZ"` | code `"40"` | `currency.Cek()` |
| — | `TestPayment_WalletNotFound` | Wallet tidak ditemukan | code `"50"` | `sql.ErrNoRows` |

### 4. Transfer — `transfer_unit_test.go` (9 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| TR-01 | `TestTransfer_TR01_Standard` | A→B transfer 300 dari saldo 1000 | A = `"700.00"`, B = `"300.00"` | `exec.Wallet().Transfer()` |
| TR-02 | `TestTransfer_TR02_DestinationNotFound` | Wallet tujuan tidak ada | code `"50"`, saldo A tidak berubah | `foundCount != 2` |
| TR-02b | `TestTransfer_TR02b_CrossCurrency` | A punya IDR, B punya USD | code `"50"` | Lookup lintas mata uang gagal |
| TR-03 | `TestTransfer_TR03_ExceedBalance` | Transfer melebihi saldo | code `"50"`, saldo tidak berubah | Constraint `wallet_cek_balance` |
| — | `TestTransfer_SelfTransfer` | Transfer ke diri sendiri (A→A) | code `"50"`, pesan `"diri sendiri"` | Validasi self-transfer |
| — | `TestTransfer_SuspendedSource` | Wallet asal dibekukan | code `"50"`, pesan `"dibekukan"` | Trigger `TRANSFER_OUT` |
| — | `TestTransfer_SuspendedDestination` | Wallet tujuan dibekukan | code `"50"`, pesan `"dibekukan"` | Trigger `TRANSFER_IN` |
| — | `TestTransfer_MissingFields` | Field `beneﬁciaryAccountNo` kosong | code `"40"` | Validasi field |
| — | `TestTransfer_ZeroAmount` | Amount `"0.00"` | code `"50"` | `validasiAmount()` |

### 5. Query Wallet — `query_unit_test.go` (5 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| W-05 | `TestQuery_W05_SuspendedStatus` | Query wallet yang dibekukan | status `"SUSPENDED"`, saldo tetap | `exec.Wallet().Status()` |
| W-05b | `TestQuery_W05b_MultiCurrency` | Query 3 wallet (EUR, IDR, USD) | 3 wallet, urutan alfabet | `ORDER BY currency` |
| W-05c | `TestQuery_W05c_WalletNotFound` | Owner ada tapi belum punya wallet | code `"50"`, pesan `"tidak ditemukan"` | `len(wallets) == 0` |
| W-05d | `TestQuery_W05d_MissingField` | Field `sourceAccountNo` kosong | code `"40"` | Validasi field |
| W-05e | `TestQuery_W05e_OwnerIDInResult` | Result mengandung `owner_id` | `result.owner_id == ownerID` | Response `wallets.Query` |

### 6. Audit Ledger — `ledger_unit_test.go` (4 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| LD-01 | `TestLedger_LD01_SumEqualsBalance` | SUM(topup) − SUM(payment) == saldo | Cocok | `core.mutasi` + `core.wallets` |
| LD-02 | `TestLedger_LD02_ImmutableMutasi` | UPDATE pada tabel mutasi → ditolak | Error `"MUTASI:IMMUTABLE"` | Trigger `trg_prevent_update_mutasi` |
| LD-03 | `TestLedger_LD03_CompletenessAfterTransfer` | Jumlah record mutasi benar setelah operasi campuran | A=3, B=1 | `countMutasi()` |
| LD-03b | `TestLedger_LD03b_LastMutasiBalanceMatch` | `current_balance` mutasi terakhir == saldo wallet | Cocok | `mutasi.current_balance` |

### 7. Concurrency — `concurrency_unit_test.go` (5 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| SYS-01 | `TestConcurrency_SYS01_ConcurrentPayments` | 10 goroutine × 10.00 payment bersamaan | Semua berhasil, saldo = 0 | `UPDATE...RETURNING` race-safe |
| SYS-02 | `TestConcurrency_SYS02_MixedOperations` | 5 topup + 5 payment bersamaan | saldo `"750.00"` | Isolasi transaksi konkuren |
| SYS-03 | `TestConcurrency_SYS03_ReadAfterWrite` | Topup lalu query langsung | saldo `"123.45"` | Konsistensi baca setelah tulis |
| SYS-04 | `TestConcurrency_SYS04_UnorderedTransactions` | Transfer dua arah (A↔B) | Total = `"2000.00"` (konservasi uang) | Konservasi dana |
| SYS-05 | `TestConcurrency_SYS05_ConcurrentTransfers` | Transfer melingkar 3 pihak | Total = `"1500.00"` (konservasi uang) | `SELECT...FOR UPDATE` locking |

### 8. Owner — `owner_unit_test.go` (3 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| OWN-01 | `TestOwner_Create_Success` | Buat owner berhasil | code `"00"`, `owner_id` format `USER\d+` | `owner.Create` → `exec.Owner().Create()` |
| OWN-02 | `TestOwner_Create_ResponseStructure` | Verifikasi struktur response lengkap | `txID` tidak kosong, `result.owner_id` ada | `middleware.HTTP200` |
| OWN-03 | `TestOwner_Create_DBFailure` | Error database saat buat owner | code `"01"`, pesan `"Failed to create owner"`, result `nil` | `owner.Create` → error path |

> **Teknik OWN-03:** Menggunakan temporary trigger `BEFORE INSERT` pada tabel `core.owner` yang memicu `RAISE EXCEPTION`. Trigger otomatis di-drop setelah test selesai (`defer`).

### 9. Routing & Middleware — `routing_unit_test.go` (3 test)

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| RT-01 | `TestRouting_UnknownProcessor` | POST `/core/unknown` (processor tidak ada) | HTTP 404, code `"40"` | `routes.Service` → `processor.Handler()` |
| RT-02 | `TestRouting_UnknownMethod` | POST `/core/wallets/unknown` (method tidak ada) | HTTP 404, code `"40"` | `routes.Service` → `processor.Handler()` |
| RT-03 | `TestRouting_HealthEndpoint` | GET `/_health` | HTTP 200, body `"I'm Ok"` | `routes.Health` |

### 10. Idempotency / Deteksi Duplikat — `idempotency_unit_test.go` (5 test)

> **Prasyarat:** Memcached harus berjalan di `localhost:11511` (via Docker Compose).  
> Test akan otomatis **di-skip** jika Memcached tidak tersedia.

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| ID-01 | `TestIdempotency_DuplicateTopup` | Topup identik 2x (owner, amount, currency sama) | Pertama: code `"00"`. Kedua: HTTP 425, code `"83"`, saldo tidak bertambah | `cache.AcquireLock()` → `ErrDuplicateRequest` |
| ID-02 | `TestIdempotency_DuplicatePayment` | Payment identik 2x | Pertama: code `"00"`. Kedua: HTTP 425, code `"83"` | `cache.AcquireLock()` → `ErrDuplicateRequest` |
| ID-03 | `TestIdempotency_DuplicateTransfer` | Transfer identik 2x | Pertama: code `"00"`. Kedua: HTTP 425, code `"83"`, saldo tidak berubah | `cache.AcquireLock()` → `ErrDuplicateRequest` |
| ID-04 | `TestIdempotency_DifferentAmountsNotDuplicate` | Topup 100 lalu topup 200 (beda amount) | Keduanya berhasil, saldo `"300.00"` | DUPLICATE_KEY berbeda → bukan duplikat |
| ID-05 | `TestIdempotency_LockExpiresAfterTTL` | Lock kadaluarsa → retry berhasil | Diblokir saat lock aktif, berhasil setelah kadaluarsa | `memcache.Item.Expiration` → TTL |

### 11. Edge Case & Boundary — `edge_case_unit_test.go` (21 test)

#### Mata Uang

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| EC-01 | `TestEdge_CurrencyLowercase` | Currency `"idr"` (huruf kecil) | code `"40"` — map ISO 4217 case-sensitive | `currency.Cek()` |

#### Format Amount

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| EC-02 | `TestEdge_AmountWithoutDecimals` | Amount `"100"` tanpa desimal | code `"00"`, saldo `"100.00"` | `decimal.NewFromString()` |
| EC-03 | `TestEdge_AmountExactMinUnit` | Amount `"0.01"` (batas minimum) | code `"00"`, saldo `"0.01"` | `validasiAmount()` → `minUnit` |
| EC-04 | `TestEdge_AmountWithComma` | Amount `"1,000.00"` (pakai koma) | code `"40"` — format tidak valid | `req.ParseAmount()` |

#### Keamanan

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| EC-05 | `TestEdge_UnicodeFieldBeneficiary` | Field `beneficiaryAccountNo` ASCII vs `beneﬁciaryAccountNo` Unicode (U+FB01) | Verifikasi perilaku — ASCII tidak cocok dengan struct tag Unicode | `WalletRequest.BeneﬁciaryAccountNo` |
| EC-06 | `TestEdge_SQLInjectionSourceAccountNo` | sourceAccountNo berisi `"'; DROP TABLE wallets; --"` | Query parameterized → aman, tidak terjadi injection | `db.Conn().ExecContext` |

#### Siklus Hidup Wallet

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| EC-07 | `TestEdge_ReactivationLifecycle` | ACTIVE → Bekukan → Gagal topup → Aktifkan → Berhasil topup | Siklus lengkap berfungsi | `Suspended()` + `Activate()` |

#### Payment — Edge Case

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| EC-08 | `TestEdge_PaymentRounding` | Payment `"12.345"` dibulatkan | saldo `"87.65"` (100 − 12.35) | `validasiAmount()` → `Round(2)` |
| EC-09 | `TestEdge_PaymentBelowMinUnit` | Payment `"0.001"` | code `"50"` — di bawah unit minimum | `validasiAmount()` → `minUnit` |
| EC-10 | `TestEdge_PaymentInvalidAmountFormat` | Payment `"abc"` | code `"40"` — format tidak valid | `req.ParseAmount()` |
| EC-11 | `TestEdge_PaymentLargeAmount` | Payment 999.999.999 dari saldo 1 Miliar | saldo `"1.00"` | Presisi `NUMERIC(19,4)` |

#### Transfer — Edge Case

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| EC-12 | `TestEdge_TransferNegativeAmount` | Transfer `"-100.00"` | code `"50"` | `validasiAmount()` |
| EC-13 | `TestEdge_TransferInvalidAmountFormat` | Transfer `"abc"` | code `"40"` | `req.ParseAmount()` |
| EC-14 | `TestEdge_TransferInvalidCurrency` | Transfer currency `"XXX"` | code `"40"` | `currency.Cek()` |
| EC-15 | `TestEdge_TransferBelowMinUnit` | Transfer `"0.001"` | code `"50"` | `validasiAmount()` → `minUnit` |
| EC-16 | `TestEdge_TransferRounding` | Transfer `"12.345"` dibulatkan | Saldo mencerminkan jumlah yang dibulatkan | `validasiAmount()` → `Round(2)` |
| EC-17 | `TestEdge_TransferExactBalance` | Transfer seluruh saldo | Pengirim = `"0.00"` | Constraint `wallet_cek_balance` |
| EC-18 | `TestEdge_TransferBothSuspended` | Wallet asal + tujuan keduanya dibekukan | code `"50"` | Trigger: mana yang terpicu duluan |

#### Audit Mutasi

| # | Fungsi Test | Skenario | Hasil yang Diharapkan | Merujuk ke |
|---|---|---|---|---|
| EC-19 | `TestEdge_MutasiAudit_Topup` | Verifikasi record mutasi setelah topup | Record ada, field sesuai | `core.mutasi` |
| EC-20 | `TestEdge_MutasiAudit_Payment` | Verifikasi record mutasi setelah payment | Record ada, field sesuai | `core.mutasi` |
| EC-21 | `TestEdge_MutasiAudit_Transfer` | Verifikasi 2 record mutasi setelah transfer | Record OUT + IN ada | `core.mutasi` |

---

## 📊 Ringkasan Test per File

| File | Jumlah | Kategori |
|---|---|---|
| `create_unit_test.go` | 6 | Pembuatan wallet |
| `topup_unit_test.go` | 12 | Transaksi topup |
| `payment_unit_test.go` | 9 | Transaksi payment |
| `transfer_unit_test.go` | 9 | Transaksi transfer |
| `query_unit_test.go` | 5 | Query status wallet |
| `ledger_unit_test.go` | 4 | Audit & integritas ledger |
| `concurrency_unit_test.go` | 5 | Concurrency & race condition |
| `owner_unit_test.go` | 3 | Endpoint owner |
| `routing_unit_test.go` | 3 | Routing & middleware |
| `idempotency_unit_test.go` | 5 | Deteksi duplikat (Memcached) |
| `edge_case_unit_test.go` | 21 | Edge case & boundary |
| **Total** | **82** | |

---

## 🏷️ Kode Response

| Kode | Arti | Contoh Penyebab |
|---|---|---|
| `"00"` | Berhasil | Operasi sukses |
| `"01"` | Gagal (owner) | Error database saat buat owner |
| `"40"` | Format tidak valid | Field kosong, currency salah, format amount salah |
| `"50"` | Gagal (wallet/transaksi) | Saldo tidak cukup, wallet dibekukan, wallet tidak ditemukan |
| `"83"` | Transaksi duplikat | Idempotency lock aktif di Memcached |
| `"500"` | Error internal | Error tidak tertangani |

---

## ✅ Status Coverage

Seluruh skenario yang teridentifikasi sudah ter-cover oleh test:

| Kategori | Jumlah | Status |
|---|---|---|
| Pembuatan wallet (happy & error path) | 6 | ✅ Ter-cover |
| Topup (normal, validasi, boundary) | 12 | ✅ Ter-cover |
| Payment (normal, validasi, boundary) | 9 | ✅ Ter-cover |
| Transfer (normal, validasi, edge case) | 9 | ✅ Ter-cover |
| Query wallet | 5 | ✅ Ter-cover |
| Audit ledger & integritas data | 4 | ✅ Ter-cover |
| Concurrency & race condition | 5 | ✅ Ter-cover |
| Owner endpoint (termasuk error path) | 3 | ✅ Ter-cover |
| Routing & middleware | 3 | ✅ Ter-cover |
| Idempotency / deteksi duplikat | 5 | ✅ Ter-cover |
| Edge case & boundary | 21 | ✅ Ter-cover |
| **Total** | **82** | **✅ Semua ter-cover** |
