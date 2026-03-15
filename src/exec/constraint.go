package exec

import (
	"app/src/utils/logger"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
)

func ErrWallet(context *fiber.Ctx, err error) error {
	log := context.Locals("logger").(*logger.CustomLogger)

	if err == nil {
		return nil
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr
	}

	msg := err.Error()

	// TRIGGER
	if strings.Contains(msg, "MUTASI:SUSPENDED:") && strings.Contains(msg, "TRANSFER_OUT") {
		return fiber.NewError(fiber.StatusForbidden, "Transaksi ditolak: Akun wallet Asal sedang dibekukan.")
	}

	if strings.Contains(msg, "MUTASI:SUSPENDED:") && strings.Contains(msg, "TRANSFER_IN") {
		return fiber.NewError(fiber.StatusForbidden, "Transaksi ditolak: Akun wallet Tujuan sedang dibekukan.")
	}

	if strings.Contains(msg, "MUTASI:SUSPENDED:") && strings.Contains(msg, "TOPUP") || strings.Contains(msg, "PAYMENT") {
		return fiber.NewError(fiber.StatusForbidden, "Transaksi ditolak: Akun wallet sedang dibekukan.")
	}

	if strings.Contains(msg, "MUTASI:IMMUTABLE:") {
		return fiber.NewError(fiber.StatusForbidden, "Akses ditolak: Histori mutasi tidak dapat diubah.")
	}

	var pqErr *pgconn.PgError
	if errors.As(err, &pqErr) {
		switch pqErr.Code {

		// ------------------------------------------
		// State 23514: CHECK CONSTRAINT VIOLATION
		// ------------------------------------------
		case "23514":
			switch pqErr.ConstraintName {
			case "wallet_cek_balance":
				return fiber.NewError(fiber.StatusBadRequest, "Transaksi gagal: Saldo wallet tidak mencukupi.")

			case "mutasi_amount_check":
				return fiber.NewError(fiber.StatusBadRequest, "Validasi gagal: Nominal transaksi harus lebih besar dari 0.")

			case "mutasi_balance_check":
				return fiber.NewError(fiber.StatusBadRequest, "Transaksi gagal: Saldo akhir pada histori mutasi tidak boleh minus.")

			default:
				return fiber.NewError(fiber.StatusBadRequest, "Validasi data gagal. Violation Check.")
			}

		// ------------------------------------------
		// State 23505: UNIQUE CONSTRAINT VIOLATION
		// ------------------------------------------
		case "23505":
			switch pqErr.ConstraintName {
			case "mutasi_unique_trx_per_wallet":
				return fiber.NewError(fiber.StatusConflict, "Mutasi duplicate: ID transaksi ini sudah pernah diproses untuk wallet tersebut.")

			case "wallet_unique_currency_per_owner":
				return fiber.NewError(fiber.StatusConflict, "Wallet duplicate: Pengguna sudah memiliki wallet dengan mata uang tersebut.")

			default:
				return fiber.NewError(fiber.StatusConflict, "Terjadi duplicate data pada sistem.")
			}

		// ------------------------------------------
		// State 23503: FOREIGN KEY VIOLATION
		// ------------------------------------------
		case "23503":
			switch pqErr.ConstraintName {
			case "wallets_owner_id_fkey":
				return fiber.NewError(fiber.StatusNotFound, "Invalid parameter: Owner ID tidak ditemukan.")

			default:
				return fiber.NewError(fiber.StatusNotFound, "Invalid parameter/ID Tidak ditemukan.")
			}
		}
	}

	log.Error(err, "[CRITICAL DB ERROR]")

	return fiber.NewError(fiber.StatusInternalServerError, "Terjadi kesalahan sistem internal. Silakan coba beberapa saat lagi.")
}
