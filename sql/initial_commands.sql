BEGIN;
CREATE TABLE currency_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);
CREATE TABLE currencies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(10) NOT NULL UNIQUE,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    type_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (type_id) REFERENCES currency_types(id)
);
-- Индекс на поле updated_at
CREATE INDEX idx_currencies_updated_at on currencies(updated_at);
-- Функция для обновления поля updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- Триггер на обновление поля updated_at в записях currencies
CREATE TRIGGER currency_update_updated_at BEFORE
UPDATE ON currencies FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- Добавление базовых валют
INSERT INTO currency_types (name)
VALUES ('FIAT');
INSERT INTO currency_types (name)
VALUES ('CRYPTO');
INSERT INTO currencies (name, code, is_enabled, type_id)
VALUES ('Euro', 'EUR', true, 1),
    ('US Dollar', 'USD', true, 1),
    ('Chinese Yuan', 'CNY', true, 1),
    ('Tether', 'USDT', true, 2),
    ('USD Coin', 'USDC', true, 2),
    ('Ethereum', 'ETH', true, 2);
-- notification to listen
CREATE OR REPLACE FUNCTION notify_currency_change() RETURNS trigger AS $$
DECLARE notification JSON;
BEGIN IF (
    TG_OP = 'INSERT'
    OR TG_OP = 'UPDATE'
) THEN notification := json_build_object(
    'operation',
    TG_OP,
    'currency',
    row_to_json(NEW)
);
ELSIF (TG_OP = 'DELETE') THEN notification := json_build_object(
    'operation',
    'DELETE',
    'currency',
    row_to_json(OLD)
);
END IF;
PERFORM pg_notify('currency_events', notification::text);
-- Return the appropriate row type
IF (TG_OP = 'DELETE') THEN RETURN OLD;
ELSE RETURN NEW;
END IF;
END;
$$ LANGUAGE plpgsql;
-- trigger for update/delete/insert
CREATE OR REPLACE TRIGGER trigger_notify_currency_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON currencies FOR EACH ROW EXECUTE FUNCTION notify_currency_change();
COMMIT;