package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/nfedorov/port_server/internal/model"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS allocations (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			app         TEXT    NOT NULL,
			instance    TEXT    NOT NULL,
			service     TEXT    NOT NULL,
			port        INTEGER NOT NULL UNIQUE,
			created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
			UNIQUE(app, instance, service, port)
		)
	`)
	return err
}

func (s *SQLiteStore) Allocate(req model.AllocateRequest, portMin, portMax int) (*model.Allocation, error) {
	port := req.Port

	if port == 0 {
		var err error
		port, err = s.findFreePort(portMin, portMax)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO allocations (app, instance, service, port, created_at) VALUES (?, ?, ?, ?, ?)`,
		req.App, req.Instance, req.Service, port, now.Format(time.DateTime),
	)
	if err != nil {
		// Check if port is taken by trying to look it up.
		existing, lookupErr := s.GetByPort(port)
		if lookupErr == nil && existing != nil {
			return existing, ErrPortTaken
		}
		return nil, err
	}

	id, _ := res.LastInsertId()
	return &model.Allocation{
		ID:        id,
		App:       req.App,
		Instance:  req.Instance,
		Service:   req.Service,
		Port:      port,
		CreatedAt: now,
	}, nil
}

func (s *SQLiteStore) findFreePort(portMin, portMax int) (int, error) {
	rows, err := s.db.Query(
		`SELECT port FROM allocations WHERE port >= ? AND port <= ? ORDER BY port`,
		portMin, portMax,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	used := make(map[int]bool)
	for rows.Next() {
		var p int
		if err := rows.Scan(&p); err != nil {
			return 0, err
		}
		used[p] = true
	}

	for p := portMin; p <= portMax; p++ {
		if !used[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free ports in range %d-%d", portMin, portMax)
}

func (s *SQLiteStore) List(f Filter) ([]model.Allocation, error) {
	query := `SELECT id, app, instance, service, port, created_at FROM allocations WHERE 1=1`
	args := []any{}

	if f.App != "" {
		query += ` AND app = ?`
		args = append(args, f.App)
	}
	if f.Instance != "" {
		query += ` AND instance = ?`
		args = append(args, f.Instance)
	}
	if f.Service != "" {
		query += ` AND service = ?`
		args = append(args, f.Service)
	}
	if f.Port != 0 {
		query += ` AND port = ?`
		args = append(args, f.Port)
	}
	query += ` ORDER BY id`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allocs []model.Allocation
	for rows.Next() {
		var a model.Allocation
		var createdAt string
		if err := rows.Scan(&a.ID, &a.App, &a.Instance, &a.Service, &a.Port, &createdAt); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
		allocs = append(allocs, a)
	}
	return allocs, nil
}

func (s *SQLiteStore) GetByPort(port int) (*model.Allocation, error) {
	var a model.Allocation
	var createdAt string
	err := s.db.QueryRow(
		`SELECT id, app, instance, service, port, created_at FROM allocations WHERE port = ?`, port,
	).Scan(&a.ID, &a.App, &a.Instance, &a.Service, &a.Port, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	return &a, nil
}

func (s *SQLiteStore) DeleteByID(id int64) error {
	res, err := s.db.Exec(`DELETE FROM allocations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) DeleteByFilter(f Filter) (int64, error) {
	query := `DELETE FROM allocations WHERE 1=1`
	args := []any{}

	if f.App != "" {
		query += ` AND app = ?`
		args = append(args, f.App)
	}
	if f.Instance != "" {
		query += ` AND instance = ?`
		args = append(args, f.Instance)
	}
	if f.Service != "" {
		query += ` AND service = ?`
		args = append(args, f.Service)
	}
	if f.Port != 0 {
		query += ` AND port = ?`
		args = append(args, f.Port)
	}

	// Safety: require at least one filter
	if len(args) == 0 {
		return 0, fmt.Errorf("at least one filter is required for delete")
	}

	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
