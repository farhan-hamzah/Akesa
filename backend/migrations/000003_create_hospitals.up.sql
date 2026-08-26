CREATE TABLE hospitals (
    id UUID PRIMARY KEY,

    name VARCHAR(255) NOT NULL,
    code VARCHAR(100) NOT NULL UNIQUE,

    address TEXT,
    phone VARCHAR(30),

    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hospitals_code
    ON hospitals(code);