package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Client struct {
	db *sql.DB
}

func NewClient(pathToDB string) (Client, error) {
	db, err := sql.Open("sqlite3", pathToDB)
	if err != nil {
		return Client{}, err
	}
	c := Client{db}
	err = c.autoMigrate()
	if err != nil {
		return Client{}, err
	}
	return c, nil

}

func (c *Client) autoMigrate() error {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		password TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL
	);
	`
	_, err := c.db.Exec(userTable)
	if err != nil {
		return err
	}
	refreshTokenTable := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		token TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		revoked_at TIMESTAMP,
		user_id TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	`
	_, err = c.db.Exec(refreshTokenTable)
	if err != nil {
		return err
	}

	videoTable := `
	CREATE TABLE IF NOT EXISTS videos (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		title TEXT NOT NULL,
		description TEXT,
		thumbnail_url TEXT,
		video_url TEXT TEXT,
		user_id INTEGER,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	`
	_, err = c.db.Exec(videoTable)
	if err != nil {
		return err
	}
	return nil
}

func (c Client) Reset() error {

	qbDeleteQueries := map[string]string{
		"refresh_tokens": "DELETE FROM refresh_tokens",
		"users":          "DELETE FROM users",
		"videos":         "DELETE FROM videos",
	}

	for tableName, deleteQuery := range qbDeleteQueries {
		if _, err := c.db.Exec(deleteQuery); err != nil {
			return fmt.Errorf("failed to reset table %s: %w", tableName, err)
		}
	}
	return nil
}
