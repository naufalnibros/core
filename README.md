# Core Wallet

API Service Core Wallet. Mendukung multi-currency (ISO 4217), transaksi topup/payment/transfer, deteksi duplikat via Memcached, dan audit trail (mutasi immutable).

---

## Tech Stack

| Komponen | Teknologi |
|---|---|
| Bahasa | Go 1.24 |
| Framework | Fiber v2 |
| Database | PostgreSQL 16 |
| Cache | Memcached 1.6 |
| Driver DB | pgx/v5 + sqlx |
| Presisi | shopspring/decimal — `NUMERIC(19,4)` |
| JSON Encoder | goccy/go-json |

---

## Requirements

- Go v1.24+
- Docker & Docker Compose

---

## Quick Start

```bash
# Clone & cd ke direktori
cd core

# Setup .env
cp .env.example .env

go mod download
go mod tidy
# go build
# ./app

# Jalankan semua service (via docker)
docker compose up -d

# Verifikasi service
docker compose ps

# Health check
curl http://localhost:11082/core/_health
# Response: I'm Ok
```

---

## Daftar Endpoint

| # | Endpoint | Deskripsi |
|---|---|---|
| 1 | `GET /core` | Homepage — daftar endpoint |
| 2 | `GET /core/_health` | Health check |
| 3 | `POST /core/owner` | Registrasi owner baru |
| 4 | `POST /core/wallets` | Membuat wallet baru |
| 5 | `POST /core/wallets/topup` | Top up saldo wallet |
| 6 | `POST /core/wallets/payment` | Pembayaran dari wallet |
| 7 | `POST /core/wallets/transfer` | Transfer antar wallet |
| 8 | `POST /core/wallets/query` | Cek saldo & informasi wallet |

---

## Struktur Response

Semua endpoint (kecuali `_health`) mengembalikan format JSON yang konsisten:

```json
{
  "attribute": {
    "txID": "2026031522170249710000",
    "code": "00",
    "message": "Pesan deskriptif.",
    "errormsg": ""
  },
  "result": null
}
```

| Field | Tipe | Keterangan |
|---|---|---|
| `attribute.txID` | `string` | ID transaksi unik (timestamp-based) |
| `attribute.code` | `string` | Response Code — lihat [Response Code](#response-code) |
| `attribute.message` | `string` | Pesan deskriptif untuk end-user |
| `attribute.errormsg` | `string` | Detail error teknis (hanya muncul di mode `dev`) |
| `result` | `object\|null` | Data hasil operasi (jika ada) |

### Response Code

| Kode | HTTP Status | Arti |
|---|---|---|
| `"00"` | 200 | Berhasil |
| `"01"` | 200 | Gagal membuat owner |
| `"40"` | 200 | Format / validasi tidak valid |
| `"50"` | 200 | Operasi gagal (constraint / business logic) |
| `"83"` | 425 | Transaksi duplikat (idempotency) |

---

## Dokumentasi Endpoint

### 1. Homepage

Menampilkan halaman HTML berisi daftar endpoint yang tersedia.

```
GET /core
```

---

### 2. Health Check

Memeriksa apakah service berjalan.

```
GET /core/_health
```

**Response:**

```
I'm Ok
```

---

### 3. Registrasi Owner

Membuat owner baru. Setiap owner mendapat ID unik format `USERxxxx`.

```
POST /core/owner
```

**Request Body:** Tidak ada (kosong).

**Contoh:**

```bash
curl --request POST \
  --url http://localhost:11082/core/owner
```

**Response Sukses:**

```json
{
  "attribute": {
    "txID": "2026031522165135110000",
    "code": "00",
    "message": "Owner created successfully.",
    "errormsg": ""
  },
  "result": {
    "owner_id": "USER1471"
  }
}
```

**Response Gagal:**

| Kode | Kondisi | Message |
|---|---|---|
| `"01"` | Error database | `"Failed to create owner"` |

---

### 4. Membuat Wallet

Membuat wallet baru untuk owner yang sudah terdaftar. Satu owner bisa memiliki banyak wallet dengan mata uang berbeda, tetapi hanya **satu wallet per mata uang**.

```
POST /core/wallets
Content-Type: application/json
```

**Request Body:**

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `sourceAccountNo` | `string` | ✅ | ID owner (contoh: `"USER1471"`) |
| `currency` | `string` | ✅ | Kode mata uang ISO 4217, case-sensitive (contoh: `"IDR"`, `"USD"`) |

**Contoh:**

```bash
curl --request POST \
  --url http://localhost:11082/core/wallets \
  --header 'Content-Type: application/json' \
  --data '{
    "sourceAccountNo": "USER1471",
    "currency": "IDR"
}'
```

**Response Sukses:**

```json
{
  "attribute": {
    "txID": "2026031522170249710000",
    "code": "00",
    "message": "Wallet created successfully.",
    "errormsg": ""
  },
  "result": null
}
```

**Response Gagal:**

| Kode | Kondisi | Message |
|---|---|---|
| `"40"` | Field kosong | `"Invalid Format: Field 'sourceAccountNo' dan 'currency' wajib di isi."` |
| `"40"` | Currency tidak valid | `"Invalid Currency: Kode mata uang 'ABC' tidak valid (ISO 4217)."` |
| `"50"` | Wallet duplikat | `"Wallet duplicate: Pengguna sudah memiliki wallet dengan mata uang tersebut."` |
| `"50"` | Owner ID tidak ditemukan | `"Owner ID tidak ditemukan."` |

---

### 5. Top Up Saldo

Menambahkan saldo ke wallet. Amount akan dibulatkan ke 2 desimal (`Round(2)`). Deteksi duplikat aktif via Memcached (TTL 10 detik).

```
POST /core/wallets/topup
Content-Type: application/json
```

**Request Body:**

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `sourceAccountNo` | `string` | ✅ | ID owner |
| `currency` | `string` | ✅ | Kode mata uang ISO 4217 |
| `amount` | `string` | ✅ | Jumlah topup (min `"0.01"`, format angka desimal) |

**Contoh:**

```bash
curl --request POST \
  --url http://localhost:11082/core/wallets/topup \
  --header 'Content-Type: application/json' \
  --data '{
    "sourceAccountNo": "USER1471",
    "amount": "1000.00",
    "currency": "IDR"
}'
```

**Response Sukses:**

```json
{
  "attribute": {
    "txID": "2026031522170999610000",
    "code": "00",
    "message": "Topup Successfull. Saldo berhasil ditambahkan sebesar 1000 IDR",
    "errormsg": ""
  },
  "result": null
}
```

**Response Gagal:**

| Kode | HTTP | Kondisi | Message |
|---|---|---|---|
| `"40"` | 200 | Field kosong | `"Invalid Format: Field 'sourceAccountNo', 'currency', dan 'amount' wajib diisi."` |
| `"40"` | 200 | Currency tidak valid | `"Invalid Currency: Kode mata uang '...' tidak valid (ISO 4217)."` |
| `"50"` | 200 | Amount ≤ 0 | `"Invalid Amount: tidak boleh 0.00 atau negatif"` |
| `"50"` | 200 | Amount < 0.01 | `"Invalid Amount: minimum 0.01"` |
| `"50"` | 200 | Wallet dibekukan | `"Transaksi gagal: Wallet telah dibekukan."` |
| `"50"` | 200 | Wallet tidak ditemukan | `"Wallet tidak ditemukan."` |
| `"83"` | 425 | Duplikat (idempotency) | `"Duplicate transaction."` |

---

### 6. Pembayaran

Memotong saldo wallet. Gagal jika saldo tidak mencukupi. Amount dibulatkan ke 2 desimal. Deteksi duplikat aktif.

```
POST /core/wallets/payment
Content-Type: application/json
```

**Request Body:**

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `sourceAccountNo` | `string` | ✅ | ID owner |
| `currency` | `string` | ✅ | Kode mata uang ISO 4217 |
| `amount` | `string` | ✅ | Jumlah pembayaran (min `"0.01"`) |

**Contoh:**

```bash
curl --request POST \
  --url http://localhost:11082/core/wallets/payment \
  --header 'Content-Type: application/json' \
  --data '{
    "sourceAccountNo": "USER1471",
    "amount": "250.50",
    "currency": "IDR"
}'
```

**Response Sukses:**

```json
{
  "attribute": {
    "txID": "2026031522174150310000",
    "code": "00",
    "message": "Payment Successful.",
    "errormsg": ""
  },
  "result": null
}
```

**Response Gagal:**

| Kode | HTTP | Kondisi | Message |
|---|---|---|---|
| `"40"` | 200 | Field kosong | `"Invalid Format: Field 'sourceAccountNo', 'currency', dan 'amount' wajib diisi."` |
| `"40"` | 200 | Currency tidak valid | `"Invalid Currency: ..."` |
| `"50"` | 200 | Amount ≤ 0 | `"Invalid Amount: tidak boleh 0.00 atau negatif"` |
| `"50"` | 200 | Saldo tidak cukup | `"Transaksi gagal: Saldo wallet tidak mencukupi."` |
| `"50"` | 200 | Wallet dibekukan | `"Transaksi gagal: Wallet telah dibekukan."` |
| `"50"` | 200 | Wallet tidak ditemukan | `"Wallet tidak ditemukan."` |
| `"83"` | 425 | Duplikat (idempotency) | `"Duplicate transaction."` |

---

### 7. Transfer

Transfer saldo antar wallet dengan mata uang yang sama. Gagal jika saldo pengirim tidak cukup, wallet tujuan tidak ditemukan, atau mata uang berbeda. Deteksi duplikat aktif.

```
POST /core/wallets/transfer
Content-Type: application/json
```

**Request Body:**

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `sourceAccountNo` | `string` | ✅ | ID owner pengirim |
| `beneﬁciaryAccountNo` | `string` | ✅ | ID owner penerima |
| `currency` | `string` | ✅ | Kode mata uang ISO 4217 (harus sama di kedua wallet) |
| `amount` | `string` | ✅ | Jumlah transfer (min `"0.01"`) |

**Contoh:**

```bash
curl --request POST \
  --url http://localhost:11082/core/wallets/transfer \
  --header 'Content-Type: application/json' \
  --data '{
    "sourceAccountNo": "USER1471",
    "beneﬁciaryAccountNo": "USER1472",
    "amount": "100.00",
    "currency": "IDR"
}'
```

**Response Sukses:**

```json
{
  "attribute": {
    "txID": "2026031522181317010000",
    "code": "00",
    "message": "Transfer Successful. Berhasil mentransfer dana sebesar 100 IDR ke wallet USER1472",
    "errormsg": ""
  },
  "result": null
}
```

**Response Gagal:**

| Kode | HTTP | Kondisi | Message |
|---|---|---|---|
| `"40"` | 200 | Field kosong | `"Invalid Format: ..."` |
| `"40"` | 200 | Currency tidak valid | `"Invalid Currency: ..."` |
| `"50"` | 200 | Amount ≤ 0 | `"Invalid Amount: tidak boleh 0.00 atau negatif"` |
| `"50"` | 200 | Transfer ke diri sendiri | `"Invalid Transfer: Tidak dapat melakukan transfer ke diri sendiri."` |
| `"50"` | 200 | Wallet tujuan tidak ditemukan | `"Wallet tujuan tidak ditemukan."` |
| `"50"` | 200 | Saldo tidak cukup | `"Transaksi gagal: Saldo wallet tidak mencukupi."` |
| `"50"` | 200 | Wallet asal/tujuan dibekukan | `"Transaksi gagal: Wallet telah dibekukan."` |
| `"83"` | 425 | Duplikat (idempotency) | `"Duplicate transaction."` |

---

### 8. Query Wallet

Menampilkan semua wallet milik owner beserta saldo dan statusnya.

```
POST /core/wallets/query
Content-Type: application/json
```

**Request Body:**

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `sourceAccountNo` | `string` | ✅ | ID owner |

**Contoh:**

```bash
curl --request POST \
  --url http://localhost:11082/core/wallets/query \
  --header 'Content-Type: application/json' \
  --data '{
    "sourceAccountNo": "USER1471"
}'
```

**Response Sukses:**

```json
{
  "attribute": {
    "txID": "2026031522182137210000",
    "code": "00",
    "message": "Query wallet berhasil.",
    "errormsg": ""
  },
  "result": {
    "owner_id": "USER1471",
    "wallets": [
      {
        "currency": "IDR",
        "balance": "649.50",
        "status": "ACTIVE",
        "updated_at": "2026-03-15 22:18:13"
      }
    ]
  }
}
```

**Response Gagal:**

| Kode | Kondisi | Message |
|---|---|---|
| `"40"` | Field kosong | `"Invalid Format: ..."` |
| `"50"` | Wallet tidak ditemukan | `"Wallet tidak ditemukan untuk owner ini."` |

---

## Mata Uang yang Didukung

Semua kode mata uang **ISO 4217** (case-sensitive, huruf kapital). Contoh:

| Kode | Nama |
|---|---|
| `IDR` | Rupiah |
| `USD` | US Dollar |
| `EUR` | Euro |
| `GBP` | Pound Sterling |
| `JPY` | Yen |
| `SGD` | Singapore Dollar |
| `MYR` | Malaysian Ringgit |
| `AUD` | Australian Dollar |
| `CNY` | Yuan Renminbi |
| `THB` | Baht |

Daftar lengkap: [`src/utils/currency/currency.go`](src/utils/currency/currency.go) — total **161 mata uang**.

---

## Docker Compose

### Service

| Service | Container | Port | Keterangan |
|---|---|---|---|
| PostgreSQL 16 | `core-postgres` | `9832` | Database utama, schema `core` |
| Memcached 1.6 | `core-memcached` | `11511` | Cache idempotency (TTL 10s) |
| App (dev) | `core-app` | `11082` | Hot reload via volume mount |

### Perintah Umum

```bash
# Start semua service
docker compose up -d

# Stop semua service
docker compose down

# Stop & hapus volume (RESET DATABASE)
docker compose down -v

# Lihat log (snapshot)
docker compose logs --tail=100

# Lihat log service tertentu (follow)
docker compose logs -f app
docker compose logs -f postgres
docker compose logs -f memcached

# Restart service tertentu
docker compose restart app
docker compose restart postgres
docker compose restart memcached

# Rebuild image tanpa cache
docker compose build --no-cache
docker compose up -d

# Monitor resource
docker stats core-app core-postgres core-memcached
```

### Production Mode

```bash
docker compose --profile production up -d
```

---

## Database

```bash
psql -h localhost -p 9832 -U coredev -d core
# Password: coredev
```

| Parameter | Nilai |
|---|---|
| Host | `localhost` |
| Port | `9832` |
| User | `coredev` |
| Password | `coredev` |
| Database | `core` |
| Schema | `core` |

---

## Unit Test

Dokumentasi lengkap unit test: [`TESTING.md`](TESTING.md)

```bash
# Pastikan semua service berjalan
docker compose up -d

# Jalankan semua test (82 test)
go test ./tests/processor/wallets/ -v -count=1 -timeout=120s
```