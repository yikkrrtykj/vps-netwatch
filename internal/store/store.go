package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if len(push.VPSNodes) > 0 {
		var existing []model.VPSNodeStatus
		if ok, err := loadStateTx(ctx, tx, "vps_nodes", &existing); err != nil {
			return err
		} else if ok {
			push.VPSNodes = mergeVPSNodes(existing, push.VPSNodes)
		}
	}
	var egressResults []model.EgressResult
	if push.Egress != nil {
		if push.Egress.CollectorID == "" {
			push.Egress.CollectorID = push.CollectorID
		}
		if existing, ok, err := loadEgressResultsTx(ctx, tx); err != nil {
			return err
		} else if ok {
			egressResults = mergeEgress(existing, *push.Egress)
		} else {
			egressResults = []model.EgressResult{*push.Egress}
		}
	}
	var latencyResults []model.ProbeResult
	if len(push.Latency) > 0 {
		for i := range push.Latency {
			if push.Latency[i].CollectorID == "" {
				push.Latency[i].CollectorID = push.CollectorID
			}
		}
		if ok, err := loadStateTx(ctx, tx, "latency", &latencyResults); err != nil {
			return err
		} else if ok {
			latencyResults = mergeLatency(latencyResults, push.Latency)
		} else {
			latencyResults = push.Latency
		}
	}
	value, err := json.Marshal(push)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO collector_snapshots (collector_id, value, created_at) VALUES (?, ?, ?)`,
		push.CollectorID, string(value), push.Timestamp.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err := saveStateTx(ctx, tx, "last_push", push); err != nil {
		return err
	}
	connectionControllers := connectionControllers(push)
	if len(connectionControllers) > 0 {
		if err := replaceConnectionsTx(ctx, tx, connectionControllers, push.Connections); err != nil {
			return err
		}
		latest, err := latestConnectionsTx(ctx, tx, 500)
		if err != nil {
			return err
		}
		if err := saveStateTx(ctx, tx, "connections", latest); err != nil {
			return err
		}
	}
	if push.Egress != nil {
		if err := saveStateTx(ctx, tx, "egress", egressResults); err != nil {
			return err
		}
	}
	if len(push.Latency) > 0 {
		if err := saveStateTx(ctx, tx, "latency", latencyResults); err != nil {
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
	return loadStateTx(ctx, s.db, key, dest)
}

func (s *Store) LoadEgress(ctx context.Context) ([]model.EgressResult, bool, error) {
	return loadEgressResultsTx(ctx, s.db)
}

func loadStateTx(ctx context.Context, query rowQueryer, key string, dest any) (bool, error) {
	var value string
	err := query.QueryRowContext(ctx, `SELECT value FROM state WHERE key = ?`, key).Scan(&value)
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

func loadEgressResultsTx(ctx context.Context, query rowQueryer) ([]model.EgressResult, bool, error) {
	var results []model.EgressResult
	ok, err := loadStateTx(ctx, query, "egress", &results)
	if err == nil {
		return results, ok, err
	}

	var single model.EgressResult
	if singleErr := loadStateTx(ctx, query, "egress", &single); singleErr != nil {
		return nil, false, err
	}
	return []model.EgressResult{single}, true, nil
}

func (s *Store) LatestConnections(ctx context.Context, limit int) ([]model.Connection, error) {
	if limit <= 0 {
		limit = 500
	}
	return latestConnectionsTx(ctx, s.db, limit)
}

func latestConnectionsTx(ctx context.Context, query queryer, limit int) ([]model.Connection, error) {
	rows, err := query.QueryContext(ctx,
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

func replaceConnectionsTx(ctx context.Context, tx *sql.Tx, controllers []string, connections []model.Connection) error {
	seen := map[string]bool{}
	for _, controller := range controllers {
		if controller == "" || seen[controller] {
			continue
		}
		seen[controller] = true
		if _, err := tx.ExecContext(ctx, `DELETE FROM connections WHERE controller = ?`, controller); err != nil {
			return err
		}
	}
	if len(connections) == 0 {
		return nil
	}
	return upsertConnectionsTx(ctx, tx, connections)
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

func mergeEgress(existing []model.EgressResult, incoming model.EgressResult) []model.EgressResult {
	if incoming.CollectorID == "" {
		return existing
	}
	replaced := false
	out := make([]model.EgressResult, 0, len(existing)+1)
	for _, item := range existing {
		if item.CollectorID == incoming.CollectorID {
			out = append(out, incoming)
			replaced = true
			continue
		}
		out = append(out, item)
	}
	if !replaced {
		out = append(out, incoming)
	}
	return out
}

func mergeLatency(existing, incoming []model.ProbeResult) []model.ProbeResult {
	byKey := make(map[string]model.ProbeResult, len(existing)+len(incoming))
	order := make([]string, 0, len(existing)+len(incoming))
	for _, result := range existing {
		key := latencyKey(result)
		if key == "" {
			continue
		}
		byKey[key] = result
		order = append(order, key)
	}
	for _, result := range incoming {
		key := latencyKey(result)
		if key == "" {
			continue
		}
		if _, ok := byKey[key]; !ok {
			order = append(order, key)
		}
		byKey[key] = result
	}
	out := make([]model.ProbeResult, 0, len(order))
	seen := map[string]bool{}
	for _, key := range order {
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, byKey[key])
	}
	return out
}

func latencyKey(result model.ProbeResult) string {
	if result.CollectorID == "" || result.Name == "" {
		return ""
	}
	return result.CollectorID + "\x00" + result.Name + "\x00" + result.Host + "\x00" + result.Protocol + "\x00" + strconv.Itoa(result.Port)
}

func connectionControllers(push model.CollectorPush) []string {
	if len(push.ConnectionControllers) > 0 {
		return push.ConnectionControllers
	}
	seen := map[string]bool{}
	var controllers []string
	for _, conn := range push.Connections {
		if conn.Controller == "" || seen[conn.Controller] {
			continue
		}
		seen[conn.Controller] = true
		controllers = append(controllers, conn.Controller)
	}
	return controllers
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
