package scylla

import (
	"context"
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
	cluster := gocql.NewCluster(hosts...)
	cluster.Timeout = 10 * time.Second
	cluster.ConnectTimeout = 10 * time.Second

	if err := createKeyspace(ctx, cluster, keyspace); err != nil {
		return nil, err
	}

	cluster.Keyspace = keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	store := &Store{session: session}
	if err := store.Migrate(ctx, keyspace); err != nil {
		session.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	s.session.Close()
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
