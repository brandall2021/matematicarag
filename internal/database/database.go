package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(databaseURL string) (*pgxpool.Pool, error) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return pool, nil
}

func Migrate(db *pgxpool.Pool) error {
	ctx := context.Background()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255),
			role VARCHAR(50) NOT NULL DEFAULT 'STUDENT',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(500),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
			role VARCHAR(20) NOT NULL,
			content TEXT NOT NULL,
			model VARCHAR(100),
			tokens_used INTEGER DEFAULT 0,
			sources JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS documents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			filename VARCHAR(500) NOT NULL,
			original_name VARCHAR(500) NOT NULL,
			type VARCHAR(50) NOT NULL,
			size BIGINT NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			metadata JSONB,
			uploaded_by UUID REFERENCES users(id),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key VARCHAR(255) PRIMARY KEY,
			value TEXT NOT NULL,
			description TEXT,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id),
			action VARCHAR(100) NOT NULL,
			entity_type VARCHAR(100),
			entity_id UUID,
			details JSONB,
			ip_address VARCHAR(45),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS usage_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id),
			operation VARCHAR(50) NOT NULL,
			model VARCHAR(100),
			tokens_input INTEGER DEFAULT 0,
			tokens_output INTEGER DEFAULT 0,
			cost_cents INTEGER DEFAULT 0,
			duration_ms INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_documents_uploaded_by ON documents(uploaded_by)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_user ON usage_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_created ON usage_logs(created_at)`,
		`CREATE TABLE IF NOT EXISTS document_chunks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			chunk_index INTEGER NOT NULL,
			content TEXT NOT NULL,
			page INTEGER DEFAULT 0,
			section TEXT DEFAULT '',
			topic TEXT DEFAULT '',
			content_type TEXT DEFAULT 'theory',
			has_formula BOOLEAN DEFAULT false,
			has_example BOOLEAN DEFAULT false,
			has_exercise BOOLEAN DEFAULT false,
			has_solution BOOLEAN DEFAULT false,
			course_id TEXT DEFAULT '',
			unit_id TEXT DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_doc ON document_chunks(document_id)`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_content_type ON document_chunks(content_type)`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_course ON document_chunks(course_id)`,
		`ALTER TABLE document_chunks ADD COLUMN IF NOT EXISTS content_tsv tsvector`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_tsv ON document_chunks USING gin(content_tsv)`,
		`CREATE OR REPLACE FUNCTION document_chunks_tsv_trigger() RETURNS trigger AS $$
		BEGIN
			NEW.content_tsv := to_tsvector('spanish', COALESCE(NEW.content, ''));
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,
		`DROP TRIGGER IF EXISTS tsvector_update ON document_chunks`,
		`CREATE TRIGGER tsvector_update BEFORE INSERT OR UPDATE ON document_chunks
			FOR EACH ROW EXECUTE FUNCTION document_chunks_tsv_trigger()`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w\nQuery: %s", err, m[:50])
		}
	}

	log.Println("Migrations completed")
	return nil
}
