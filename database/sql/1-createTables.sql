BEGIN;

-- Users table
CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		provider VARCHAR(255) NOT NULL,
		id_by_provider VARCHAR(255) NOT NULL,
		name VARCHAR(255),
		nickname VARCHAR(255),
		email VARCHAR(255),
		location VARCHAR(255),
		description TEXT,
		access_token TEXT,
		refresh_token TEXT,
		expires_at TIMESTAMPTZ,
		picture_link TEXT,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(provider, id_by_provider)
);

-- Conversations table
CREATE TABLE conversations (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
		history JSONB NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_provider_id_by_provider ON users (provider, id_by_provider);
CREATE INDEX idx_conversations_user_id ON conversations (user_id);
CREATE INDEX idx_conversations_created_at ON conversations (created_at);

COMMIT;
