CREATE SCHEMA IF NOT EXISTS core;
SET search_path TO core;

CREATE SEQUENCE core.owner_user_seq START 1 MAXVALUE 999999;

CREATE OR REPLACE FUNCTION core.generate_owner_user()
RETURNS VARCHAR(10) AS $$
BEGIN
    RETURN 'USER' || nextval('core.owner_user_seq')::text;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE core.owner (
    owner_id VARCHAR(10) PRIMARY KEY DEFAULT core.generate_owner_user()
);

INSERT INTO core.owner (owner_id)
SELECT core.generate_owner_user()
FROM generate_series(1, 2000);

CREATE TYPE core.wallet_status AS ENUM ('ACTIVE', 'SUSPENDED');

CREATE TABLE core.wallets (
    wallet_id BIGSERIAL PRIMARY KEY,
    owner_id VARCHAR(10) NOT NULL REFERENCES core.owner(owner_id),
    currency CHAR(3) NOT NULL, 
    balance NUMERIC(19, 4) NOT NULL DEFAULT 0.0000, 
    status core.wallet_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT wallet_unique_currency_per_owner UNIQUE (owner_id, currency),
    CONSTRAINT wallet_cek_balance CHECK (balance >= 0)
);

CREATE INDEX idx_wallets_ownerid ON core.wallets(owner_id);

CREATE TABLE core.mutasi (
    mutasi_id BIGSERIAL PRIMARY KEY,
    wallet_id INT8 NOT NULL,
    transaction_id VARCHAR(64) NOT NULL,
    tipe VARCHAR(100) NOT NULL,
    amount NUMERIC(19, 4) NOT NULL, 
    current_balance NUMERIC(19, 4) NOT NULL, 
    keterangan TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT mutasi_wallets_fk FOREIGN KEY (wallet_id) REFERENCES core.wallets(wallet_id),
    CONSTRAINT mutasi_amount_cek CHECK (amount > 0),
    CONSTRAINT mutasi_balance_cek CHECK (current_balance >= 0),
    CONSTRAINT mutasi_unique_trx_per_wallet UNIQUE (transaction_id, wallet_id)
);

CREATE INDEX idx_mutasi_history ON core.mutasi (wallet_id, mutasi_id DESC);

CREATE OR REPLACE FUNCTION core.prevent_update_mutasi()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'MUTASI:IMMUTABLE:Cannot update. txID: %', NEW.transaction_id;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_update_mutasi
BEFORE UPDATE ON core.mutasi
FOR EACH ROW
EXECUTE FUNCTION core.prevent_update_mutasi();

CREATE OR REPLACE FUNCTION core.check_wallet_status()
RETURNS TRIGGER AS $$
DECLARE
    v_status core.wallet_status;
BEGIN
    -- ASUMSI JIKA TIPE MUTASI TERKAIT DENGAN STATUS SUSPENDED DITOLAK.
    -- IF NEW.tipe ILIKE 'TOPUP%' OR NEW.tipe ILIKE 'PAYMENT%' OR NEW.tipe ILIKE 'TRANSFER%' OR NEW.tipe ILIKE 'WITHDRAW%' THEN
    -- END IF;

    SELECT status INTO v_status 
    FROM core.wallets 
    WHERE wallet_id = NEW.wallet_id;

    IF v_status = 'SUSPENDED' THEN
        RAISE EXCEPTION 'MUTASI:SUSPENDED:Wallet ID:% is SUSPENDED. Cannot process % operation. txID: %', 
            NEW.wallet_id, NEW.tipe, NEW.transaction_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_check_wallet_status_mutasi
BEFORE INSERT ON core.mutasi
FOR EACH ROW
EXECUTE FUNCTION core.check_wallet_status();

ALTER USER coredev SET search_path TO core, public;

GRANT ALL PRIVILEGES ON SCHEMA core TO coredev;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA core TO coredev;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA core TO coredev;

ALTER DEFAULT PRIVILEGES IN SCHEMA core GRANT ALL PRIVILEGES ON TABLES TO coredev;
ALTER DEFAULT PRIVILEGES IN SCHEMA core GRANT ALL PRIVILEGES ON SEQUENCES TO coredev;