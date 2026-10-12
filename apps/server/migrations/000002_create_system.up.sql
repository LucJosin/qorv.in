CREATE TABLE IF NOT EXISTS system (
    singleton_id  TINYINT NOT NULL DEFAULT 1 PRIMARY KEY,
    owner_user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    ON UPDATE CURRENT_TIMESTAMP,

    CONSTRAINT chk_system_singleton CHECK (singleton_id = 1),

    CONSTRAINT fk_system_owner_user_id
    FOREIGN KEY (owner_user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT
);
