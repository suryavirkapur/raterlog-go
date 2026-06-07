package scylla

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

type Store struct {
	session *gocql.Session
}

type Log struct {
	ChannelID    string    `json:"channel_id"`
	Timestamp    time.Time `json:"timestamp"`
	EventName    string    `json:"event_name"`
	EventPayload string    `json:"event_payload"`
	Metadata     *string   `json:"metadata"`
}

func Connect(ctx context.Context, hosts []string, keyspace string) (*Store, error) {
	if !validIdentifier(keyspace) {
		return nil, fmt.Errorf("invalid keyspace name %q", keyspace)
	}
	cluster := gocql.NewCluster(hosts...)
	cluster.Timeout = 10 * time.Second
	cluster.ConnectTimeout = 10 * time.Second

	root, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	err = root.Query(`CREATE KEYSPACE IF NOT EXISTS ` + keyspace + `
		WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '1'}
		AND durable_writes = true`).WithContext(ctx).Exec()
	root.Close()
	if err != nil {
		return nil, err
	}

	cluster.Keyspace = keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	store := &Store{session: session}
	if err := store.Migrate(ctx); err != nil {
		session.Close()
		return nil, err
	}
	return store, nil
}

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func (s *Store) Close() {
	s.session.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	return s.session.Query(`CREATE TABLE IF NOT EXISTS logs (
		channel_id text,
		timestamp timestamp,
		event_name text,
		event_payload text,
		metadata text,
		PRIMARY KEY ((channel_id), timestamp)
	) WITH CLUSTERING ORDER BY (timestamp DESC)`).WithContext(ctx).Exec()
}

func (s *Store) CreateLog(ctx context.Context, log Log) error {
	return s.session.Query(
		`INSERT INTO logs (channel_id, timestamp, event_name, event_payload, metadata)
		 VALUES (?, ?, ?, ?, ?)`,
		log.ChannelID, log.Timestamp, log.EventName, log.EventPayload, log.Metadata,
	).WithContext(ctx).Exec()
}

func (s *Store) Logs(ctx context.Context, channelID string, limit int) ([]Log, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	iter := s.session.Query(
		`SELECT channel_id, timestamp, event_name, event_payload, metadata
		 FROM logs
		 WHERE channel_id = ?
		 LIMIT ?`,
		channelID, limit,
	).WithContext(ctx).Iter()

	logs := []Log{}
	var log Log
	for iter.Scan(&log.ChannelID, &log.Timestamp, &log.EventName, &log.EventPayload, &log.Metadata) {
		item := log
		logs = append(logs, item)
		log = Log{}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return logs, nil
}
