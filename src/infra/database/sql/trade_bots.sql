CREATE TABLE trade_bots
(
    id            VARCHAR(36) PRIMARY KEY,
    symbol        VARCHAR(20)      NOT NULL,
    quantity      DOUBLE PRECISION NOT NULL,
    strategy_name VARCHAR(50)      NOT NULL,
    status        VARCHAR(20)      NOT NULL,
    created_at    TIMESTAMP        NOT NULL
);
