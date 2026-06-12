CREATE TABLE IF NOT EXISTS ip_blocks (
    ip_address VARCHAR(45) PRIMARY KEY,
    reason VARCHAR(255) NOT NULL,
    blocked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ip_blocks_expires_at ON ip_blocks(expires_at);
