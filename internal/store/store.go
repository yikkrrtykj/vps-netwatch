package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yikkrrtykj/vps-netwatch/internal/model"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA journal_mode = WAL;`,
		`CREATE TABLE IF NOT EXISTS state (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS collector_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collector_id TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS connections (
			controller TEXT NOT NULL,
			id TEXT NOT NULL,
			network TEXT,
			source_ip TEXT,
			source_port INTEGER,
			dest_ip TEXT,
			dest_port INTEGER,
			host TEXT,
			rule TEXT,
			rule_payload TEXT,
			chains TEXT,
			process TEXT,
			process_path TEXT,
			upload INTEGER,
			download INTEGER,
			started_at TEXT,
			updated_at TEXT,
			PRIMARY KEY (controller, id)
		);`,
	}
	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveCollectorPush(ctx context.Context, push model.CollectorPush) error {
	if push.Timestamp.IsZero() {
		push.Timestamp = time.Now().UTC()
	}
	if len(push.VPSNodes) > 0 {
		var existing []model.VPSNodeStatus
		if ok, err := s.LoadState(ctx, "vps_nodes", &existing); err != nil {
			return err
		} else if ok {
			push.VPSNodes = mergeVPSNodes(existing, push.VPSNodes)
		}
	}
	value, err := json.Marshal(push)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO collector_snapshots (collector_id, value, created_at) VALUES (?, ?, ?)`,
		push.CollectorID, string(value), push.Timestamp.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err := saveStateTx(ctx, tx, "last_push", push); err != nil {
		return err
	}
	if len(push.Connections) > 0 {
		if err := upsertConnectionsTx(ctx, tx, push.Connections); err != nil {
			return err
		}
		if err := saveStateTx(ctx, tx, "connections", push.Connections); err != nil {
			return err
		}
	}
	if push.Egress != nil {
		if err := saveStateTx(ctx, tx, "egress", push.Egress); err != nil {
			return err
		}
	}
	if len(push.Latency) > 0 {
		if err := saveStateTx(ctx, tx, "latency", push.Latency); err != nil {
			return err
		}
	}
	if len(push.VPSNodes) > 0 {
		if err := saveStateTx(ctx, tx, "vps_nodes", push.VPSNodes); err != nil {
			return err
		}
	}
	if len(push.Errors) > 0 {
		if err := saveStateTx(ctx, tx, "collector_errors", push.Errors); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SaveState(ctx context.Context, key string, value any) error {
	return saveStateTx(ctx, s.db, key, value)
}

func saveStateTx(ctx context.Context, exec execer, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx,
		`INSERT INTO state (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, string(data), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) LoadState(ctx context.Context, key string, dest any) (bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM state WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(value), dest); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) LatestConnections(ctx context.Context, limit int) ([]model.Connection, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT controller, id, network, source_ip, source_port, dest_ip, dest_port, host,
			rule, rule_payload, chains, process, process_path, upload, download, started_at, updated_at
		FROM connections
		ORDER BY updated_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Connection
	for rows.Next() {
		var conn model.Connection
		var chainsJSON, startedRaw, updatedRaw string
		if err := rows.Scan(
			&conn.Controller,
			&conn.ID,
			&conn.Network,
			&conn.SourceIP,
			&conn.SourcePort,
			&conn.DestIP,
			&conn.DestPort,
			&conn.Host,
			&conn.Rule,
			&conn.RulePayload,
			&chainsJSON,
			&conn.Process,
			&conn.ProcessPath,
			&conn.Upload,
			&conn.Download,
			&startedRaw,
			&updatedRaw,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(chainsJSON), &conn.Chains)
		conn.StartedAt, _ = time.Parse(time.RFC3339Nano, startedRaw)
		conn.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedRaw)
		out = append(out, conn)
	}
	return out, rows.Err()
}

func upsertConnectionsTx(ctx context.Context, tx *sql.Tx, connections []model.Connection) error {
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO connections (
			controller, id, network, source_ip, source_port, dest_ip, dest_port, host,
			rule, rule_payload, chains, process, process_path, upload, download, started_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(controller, id) DO UPDATE SET
			network = excluded.network,
			source_ip = excluded.source_ip,
			source_port = excluded.source_port,
			dest_ip = excluded.dest_ip,
			dest_port = excluded.dest_port,
			host = excluded.host,
			rule = excluded.rule,
			rule_payload = excluded.rule_payload,
			chains = excluded.chains,
			process = excluded.process,
			process_path = excluded.process_path,
			upload = excluded.upload,
			download = excluded.download,
			started_at = excluded.started_at,
			updated_at = excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, conn := range connections {
		chains, err := json.Marshal(conn.Chains)
		if err != nil {
			return err
		}
		if conn.StartedAt.IsZero() {
			conn.StartedAt = time.Now().UTC()
		}
		if conn.UpdatedAt.IsZero() {
			conn.UpdatedAt = time.Now().UTC()
		}
		if _, err := stmt.ExecContext(ctx,
			conn.Controller,
			conn.ID,
			conn.Network,
			conn.SourceIP,
			conn.SourcePort,
			conn.DestIP,
			conn.DestPort,
			conn.Host,
			conn.Rule,
			conn.RulePayload,
			string(chains),
			conn.Process,
			conn.ProcessPath,
			conn.Upload,
			conn.Download,
			conn.StartedAt.Format(time.RFC3339Nano),
			conn.UpdatedAt.Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}
	return nil
}

func mergeVPSNodes(existing, incoming []model.VPSNodeStatus) []model.VPSNodeStatus {
	byID := make(map[string]model.VPSNodeStatus, len(existing)+len(incoming))
	order := make([]string, 0, len(existing)+len(incoming))
	for _, node := range existing {
		if node.ID == "" {
			continue
		}
		byID[node.ID] = node
		order = append(order, node.ID)
	}
	for _, node := range incoming {
		if node.ID == "" {
			continue
		}
		if _, ok := byID[node.ID]; !ok {
			order = append(order, node.ID)
		}
		byID[node.ID] = node
	}
	out := make([]model.VPSNodeStatus, 0, len(order))
	seen := map[string]bool{}
	for _, id := range order {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, byID[id])
	}
	return out
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
