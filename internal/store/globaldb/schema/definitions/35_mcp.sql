CREATE TABLE mcp_auth_tokens (
			server_name    TEXT PRIMARY KEY,
			issuer         TEXT NOT NULL DEFAULT '',
			client_id      TEXT NOT NULL,
			scopes_json    TEXT NOT NULL DEFAULT '[]',
			access_token_ref   TEXT NOT NULL,
			refresh_token_ref  TEXT NOT NULL DEFAULT '',
			token_type     TEXT NOT NULL DEFAULT 'Bearer',
			expires_at     TEXT,
			obtained_at    TEXT NOT NULL,
			updated_at     TEXT NOT NULL
		);

CREATE INDEX idx_mcp_auth_tokens_updated_at
			ON mcp_auth_tokens(updated_at);
