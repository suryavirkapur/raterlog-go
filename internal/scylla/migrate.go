package scylla

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

const (
	keyspaceMigration       = "0001_keyspace.cql"
	migrationTableMigration = "0002_schema_migrations.cql"
)

//go:embed migrations/*.cql
var scyllaMigrations embed.FS

func createKeyspace(ctx context.Context, cluster *gocql.ClusterConfig, keyspace string) error {
	if !validIdentifier(keyspace) {
		return fmt.Errorf("invalid keyspace name %q", keyspace)
	}
	session, err := cluster.CreateSession()
	if err != nil {
		return err
	}
	defer session.Close()

	body, err := readCQL(keyspaceMigration, keyspace)
	if err != nil {
		return err
	}
	return session.Query(body).WithContext(ctx).Exec()
}

func (s *Store) Migrate(ctx context.Context, keyspace string) error {
	if err := s.applyUntracked(ctx, keyspace, migrationTableMigration); err != nil {
		return err
	}
	if err := s.recordMigration(ctx, migrationTableMigration); err != nil {
		return err
	}

	entries, err := fs.ReadDir(scyllaMigrations, "migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".cql" || name == keyspaceMigration || name == migrationTableMigration {
			continue
		}
		applied, err := s.migrationApplied(ctx, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := s.applyUntracked(ctx, keyspace, name); err != nil {
			return err
		}
		if err := s.recordMigration(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) applyUntracked(ctx context.Context, keyspace, name string) error {
	body, err := readCQL(name, keyspace)
	if err != nil {
		return err
	}
	return s.session.Query(body).WithContext(ctx).Exec()
}

func (s *Store) migrationApplied(ctx context.Context, version string) (bool, error) {
	var appliedAt time.Time
	err := s.session.Query(
		`SELECT applied_at FROM schema_migrations WHERE version = ?`,
		version,
	).WithContext(ctx).Scan(&appliedAt)
	if err == gocql.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) recordMigration(ctx context.Context, version string) error {
	return s.session.Query(
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		version, time.Now().UTC(),
	).WithContext(ctx).Exec()
}

func readCQL(name, keyspace string) (string, error) {
	body, err := scyllaMigrations.ReadFile(filepath.Join("migrations", name))
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(body), "{{keyspace}}", keyspace), nil
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
