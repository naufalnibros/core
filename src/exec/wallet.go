package exec

import (
	"app/src/utils/db"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

const MaxTxTimeout = 15 * time.Second

var minUnit = decimal.RequireFromString("0.01")

type walletService struct {
	ctx             context.Context
	txID            string
	SourceAccountNo string
	currency        string
}

func Wallet(ctx context.Context, SourceAccountNo, currency string, txID ...string) walletService {
	var parsedTxID string
	if len(txID) > 0 {
		parsedTxID = txID[0]
	}

	return walletService{
		ctx:             ctx,
		txID:            parsedTxID,
		SourceAccountNo: SourceAccountNo,
		currency:        currency,
	}
}

func (service walletService) validasiTxID() error {
	if service.txID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request: Transaction ID tidak ditemukan")
	}

	return nil
}

func (service walletService) validasiAmount(amount decimal.Decimal) (decimal.Decimal, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fiber.NewError(fiber.StatusBadRequest, "Invalid Amount: tidak boleh 0.00 atau negatif")
	}

	if amount.LessThan(minUnit) {
		return decimal.Zero, fiber.NewError(fiber.StatusBadRequest, "Invalid Amount: lebih kecil dari unit minimum (0.01)")
	}

	return amount.Round(2), nil
}

func (service walletService) Create() error {
	ctx, cancel := context.WithTimeout(service.ctx, MaxTxTimeout)
	defer cancel()

	_, err := db.Conn().ExecContext(ctx, `
		INSERT INTO core.wallets (owner_id, currency) 
		VALUES ($1, $2)
	`, service.SourceAccountNo, service.currency)

	return err
}

func (service walletService) Suspended() error {
	ctx, cancel := context.WithTimeout(service.ctx, MaxTxTimeout)
	defer cancel()

	_, err := db.Conn().ExecContext(ctx, `
		UPDATE core.wallets SET status = 'SUSPENDED' WHERE owner_id = $1 AND currency = $2
	`, service.SourceAccountNo, service.currency)

	return err
}

func (service walletService) Activate() error {
	ctx, cancel := context.WithTimeout(service.ctx, MaxTxTimeout)
	defer cancel()

	_, err := db.Conn().ExecContext(ctx, `
		UPDATE core.wallets SET status = 'ACTIVE' WHERE owner_id = $1 AND currency = $2
	`, service.SourceAccountNo, service.currency)

	return err
}

type Wallets struct {
	Currency  string `json:"currency" db:"currency"`
	Balance   string `json:"balance" db:"balance"`
	Status    string `json:"status" db:"status"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

func (service walletService) Status() ([]Wallets, error) {
	ctx, cancel := context.WithTimeout(service.ctx, MaxTxTimeout)
	defer cancel()

	var wallets []Wallets

	query := `
		SELECT 
			currency, 
			TO_CHAR(balance, 'FM999999999999990.00') AS balance, 
			status::TEXT AS status, 
			TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI:SS') AS updated_at
		FROM core.wallets 
		WHERE owner_id = $1 
		ORDER BY currency`

	err := db.Conn().SelectContext(ctx, &wallets, query, service.SourceAccountNo)
	if err != nil {
		return nil, err
	}

	if len(wallets) == 0 {
		return nil, fiber.NewError(fiber.StatusNotFound, "Wallet tidak ditemukan untuk owner ini.")
	}

	return wallets, nil
}

func (service walletService) Topup(amount decimal.Decimal, keterangan string) error {
	if err := service.validasiTxID(); err != nil {
		return err
	}

	validAmount, err := service.validasiAmount(amount)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(service.ctx, MaxTxTimeout)
	defer cancel()

	tx, err := db.Conn().BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var walletID int64
	var currentBalance decimal.Decimal

	err = tx.QueryRowContext(ctx, `
		UPDATE core.wallets 
		SET balance = balance + $1, updated_at = NOW() 
		WHERE owner_id = $2 AND currency = $3
		RETURNING wallet_id, balance
	`, validAmount.String(), service.SourceAccountNo, service.currency).Scan(&walletID, &currentBalance)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "Invalid Wallet: Wallet tidak ditemukan.")
		}

		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO core.mutasi (wallet_id, transaction_id, tipe, amount, current_balance, keterangan) 
		VALUES ($1, $2, 'TOPUP', $3, $4, $5)
	`, walletID, service.txID, validAmount.String(), currentBalance.String(), keterangan)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (service walletService) Payment(amount decimal.Decimal, keterangan string) error {
	if err := service.validasiTxID(); err != nil {
		return err
	}

	validAmount, err := service.validasiAmount(amount)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(service.ctx, MaxTxTimeout)
	defer cancel()

	tx, err := db.Conn().BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var walletID int64
	var currentBalance decimal.Decimal

	err = tx.QueryRowContext(ctx, `
		UPDATE core.wallets 
		SET balance = balance - $1, updated_at = NOW() 
		WHERE owner_id = $2 AND currency = $3
		RETURNING wallet_id, balance
	`, validAmount.String(), service.SourceAccountNo, service.currency).Scan(&walletID, &currentBalance)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "Invalid Wallet: Wallet tidak ditemukan.")
		}

		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO core.mutasi (wallet_id, transaction_id, tipe, amount, current_balance, keterangan) 
		VALUES ($1, $2, 'PAYMENT', $3, $4, $5)
	`, walletID, service.txID, validAmount.String(), currentBalance.String(), keterangan)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (service walletService) Transfer(BeneﬁciaryAccountNo string, amount decimal.Decimal, keterangan string) error {
	if err := service.validasiTxID(); err != nil {
		return err
	}

	validAmount, err := service.validasiAmount(amount)
	if err != nil {
		return err
	}

	if service.SourceAccountNo == BeneﬁciaryAccountNo {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid Transfer: Tidak dapat melakukan transfer ke diri sendiri.")
	}

	ctx, cancel := context.WithTimeout(service.ctx, MaxTxTimeout)
	defer cancel()

	tx, err := db.Conn().BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT wallet_id, owner_id 
		FROM core.wallets 
		WHERE owner_id IN ($1, $2) AND currency = $3 
		ORDER BY owner_id FOR UPDATE
	`, service.SourceAccountNo, BeneﬁciaryAccountNo, service.currency)
	if err != nil {
		return fmt.Errorf("lock wallets: %w", err)
	}
	defer rows.Close()

	var fromWalletID, toWalletID int64
	var foundCount int

	for rows.Next() {
		var wID int64
		var oID string
		if err := rows.Scan(&wID, &oID); err != nil {
			return fmt.Errorf("scan lock row: %w", err)
		}

		switch oID {
		case service.SourceAccountNo:
			fromWalletID = wID
		case BeneﬁciaryAccountNo:
			toWalletID = wID
		}
		foundCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}

	if foundCount != 2 {
		return fiber.NewError(fiber.StatusNotFound, "Invalid Wallet: Dompet asal atau tujuan tidak ditemukan.")
	}

	var fromBalance decimal.Decimal
	err = tx.QueryRowContext(ctx, `
		UPDATE core.wallets SET balance = balance - $1, updated_at = NOW() 
		WHERE wallet_id = $2 RETURNING balance
	`, validAmount.String(), fromWalletID).Scan(&fromBalance)
	if err != nil {
		return err
	}

	var toBalance decimal.Decimal
	err = tx.QueryRowContext(ctx, `
		UPDATE core.wallets SET balance = balance + $1, updated_at = NOW() 
		WHERE wallet_id = $2 RETURNING balance
	`, validAmount.String(), toWalletID).Scan(&toBalance)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO core.mutasi (wallet_id, transaction_id, tipe, amount, current_balance, keterangan) 
		VALUES ($1, $2, 'TRANSFER_OUT', $3, $4, $5)
	`, fromWalletID, service.txID, validAmount.String(), fromBalance.String(), fmt.Sprintf("%s; TO %s; %s %s", keterangan, BeneﬁciaryAccountNo, validAmount.String(), service.currency))
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO core.mutasi (wallet_id, transaction_id, tipe, amount, current_balance, keterangan) 
		VALUES ($1, $2, 'TRANSFER_IN', $3, $4, $5)
	`, toWalletID, service.txID, validAmount.String(), toBalance.String(), fmt.Sprintf("%s; FROM %s; %s %s", keterangan, service.SourceAccountNo, validAmount.String(), service.currency))
	if err != nil {
		return err
	}

	return tx.Commit()
}
