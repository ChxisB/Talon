package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "talon.db")
	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	DB.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes

	if err := migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS project (
		id TEXT PRIMARY KEY,
		worktree TEXT NOT NULL,
		vcs TEXT,
		name TEXT,
		icon_url TEXT,
		icon_url_override TEXT,
		icon_color TEXT,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		time_initialized INTEGER,
		sandboxes TEXT NOT NULL DEFAULT '[]',
		commands TEXT
	);

	CREATE TABLE IF NOT EXISTS workspace (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		branch TEXT,
		directory TEXT,
		extra TEXT,
		project_id TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE,
		time_used INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS project_directory (
		project_id TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE,
		directory TEXT NOT NULL,
		type TEXT,
		strategy TEXT,
		time_created INTEGER NOT NULL,
		PRIMARY KEY (project_id, directory)
	);

	CREATE TABLE IF NOT EXISTS session (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE,
		workspace_id TEXT REFERENCES workspace(id),
		parent_id TEXT REFERENCES session(id),
		slug TEXT NOT NULL,
		directory TEXT NOT NULL,
		path TEXT,
		title TEXT NOT NULL,
		version TEXT NOT NULL,
		share_url TEXT,
		summary_additions INTEGER,
		summary_deletions INTEGER,
		summary_files INTEGER,
		summary_diffs TEXT,
		metadata TEXT,
		cost REAL NOT NULL DEFAULT 0,
		tokens_input INTEGER NOT NULL DEFAULT 0,
		tokens_output INTEGER NOT NULL DEFAULT 0,
		tokens_reasoning INTEGER NOT NULL DEFAULT 0,
		tokens_cache_read INTEGER NOT NULL DEFAULT 0,
		tokens_cache_write INTEGER NOT NULL DEFAULT 0,
		revert TEXT,
		permission TEXT,
		agent TEXT,
		model TEXT,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		time_compacting INTEGER,
		time_archived INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_session_project ON session(project_id);
	CREATE INDEX IF NOT EXISTS idx_session_workspace ON session(workspace_id);

	CREATE TABLE IF NOT EXISTS session_message (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
		type TEXT NOT NULL,
		seq INTEGER NOT NULL,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		data TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_msg_session_seq ON session_message(session_id, seq);
	CREATE INDEX IF NOT EXISTS idx_msg_session_type_seq ON session_message(session_id, type, seq);

	CREATE TABLE IF NOT EXISTS session_input (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
		prompt TEXT NOT NULL,
		delivery TEXT NOT NULL,
		admitted_seq INTEGER NOT NULL,
		promoted_seq INTEGER,
		time_created INTEGER NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_input_admitted ON session_input(session_id, admitted_seq);

	CREATE TABLE IF NOT EXISTS credential (
		id TEXT PRIMARY KEY,
		integration_id TEXT,
		label TEXT NOT NULL,
		value TEXT NOT NULL,
		connector_id TEXT,
		method_id TEXT,
		active INTEGER DEFAULT 1,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS permission (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE,
		action TEXT NOT NULL,
		resource TEXT NOT NULL,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_perm_project_action ON permission(project_id, action, resource);

	CREATE TABLE IF NOT EXISTS account (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		url TEXT NOT NULL,
		access_token TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		token_expiry INTEGER,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS account_state (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		active_account_id TEXT REFERENCES account(id) ON DELETE SET NULL,
		active_org_id TEXT
	);

	CREATE TABLE IF NOT EXISTS data_migration (
		name TEXT PRIMARY KEY,
		time_completed INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS event_sequence (
		aggregate_id TEXT PRIMARY KEY,
		seq INTEGER NOT NULL,
		owner_id TEXT
	);

	CREATE TABLE IF NOT EXISTS event (
		id TEXT PRIMARY KEY,
		aggregate_id TEXT NOT NULL REFERENCES event_sequence(aggregate_id) ON DELETE CASCADE,
		seq INTEGER NOT NULL,
		type TEXT NOT NULL,
		data TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_event_agg_seq ON event(aggregate_id, seq);
	`
	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// Insert default project if not exists
	_, err = DB.Exec(`INSERT OR IGNORE INTO project (id, worktree, time_created, time_updated) VALUES ('default', '/tmp', 0, 0)`)
	return err
}
