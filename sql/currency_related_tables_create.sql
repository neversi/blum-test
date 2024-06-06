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
COMMIT;