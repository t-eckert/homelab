package db

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/lib/pq"
)

type Client struct {
	conn *sql.DB
}

func NewClient(connectionString string, password string) (*Client, error) {
	// Build connection string with password in URI format
	// The PGPASSWORD environment variable doesn't work with lib/pq DSN format
	connWithPassword := BuildConnectionURI(
		extractHost(connectionString),
		extractPort(connectionString),
		extractUser(connectionString),
		password,
		extractDatabase(connectionString),
	)

	fmt.Printf("DEBUG: DSN parts - host:%s port:%s user:%s db:%s\n",
		extractHost(connectionString),
		extractPort(connectionString),
		extractUser(connectionString),
		extractDatabase(connectionString))

	conn, err := sql.Open("postgres", connWithPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	err = conn.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &Client{conn: conn}, nil
}

func hidePassword(uri string) string {
	// Hide password in debug output
	return "postgresql://spark:****@..."
}

func extractHost(dsn string) string {
	// Extract host from "host=X port=Y..." format
	parts := parseDSN(dsn)
	return parts["host"]
}

func extractPort(dsn string) string {
	parts := parseDSN(dsn)
	return parts["port"]
}

func extractUser(dsn string) string {
	parts := parseDSN(dsn)
	return parts["user"]
}

func extractDatabase(dsn string) string {
	parts := parseDSN(dsn)
	return parts["dbname"]
}

func parseDSN(dsn string) map[string]string {
	parts := make(map[string]string)
	for _, pair := range splitDSN(dsn) {
		kv := splitPair(pair, '=')
		if len(kv) == 2 {
			parts[kv[0]] = kv[1]
		}
	}
	return parts
}

func splitDSN(s string) []string {
	var result []string
	var current string
	for _, char := range s {
		if char == ' ' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func splitPair(s string, sep rune) []string {
	idx := -1
	for i, char := range s {
		if char == sep {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

func (c *Client) CreateDatabase(name string) error {
	// Check if database already exists
	var exists bool
	err := c.conn.QueryRow("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)", name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if exists {
		return fmt.Errorf("database %s already exists", name)
	}

	// Create database
	_, err = c.conn.Exec(fmt.Sprintf("CREATE DATABASE %q", name))
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}

func (c *Client) DeleteDatabase(name string) error {
	// Terminate existing connections
	_, err := c.conn.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = %q
		AND pid <> pg_backend_pid()`, name))
	if err != nil {
		return fmt.Errorf("failed to terminate connections: %w", err)
	}

	// Drop database
	_, err = c.conn.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %q", name))
	if err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func BuildConnectionString(host, port, user, database string) string {
	// Don't include password in connection string - it will be set via PGPASSWORD env var
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
		host, port, user, database)
}

func BuildConnectionURI(host, port, user, password, database string) string {
	// Build PostgreSQL connection URI with URL-encoded password for use in containers
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(user),
		url.QueryEscape(password),
		host,
		port,
		database)
}
