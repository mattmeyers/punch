package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/mattmeyers/punch/database"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	Conn *sqlx.DB
}

var dbFilePath = getDBFilePath()
var dbFileDSN = fmt.Sprintf("file:%s?mode=rwc", dbFilePath)

func getDBFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err.Error())
	}

	return filepath.Join(home, ".local", "share", "punch", "punch.db")
}

func assertDataDirExists() error {
	return os.MkdirAll(filepath.Dir(dbFilePath), 0755)
}

func getDB() (DB, error) {
	err := assertDataDirExists()
	if err != nil {
		return DB{}, err
	}

	db, err := openDB(dbFileDSN)
	if err != nil {
		return DB{}, err
	}

	driver, err := sqlite3.WithInstance(db.Conn.DB, &sqlite3.Config{})
	if err != nil {
		return DB{}, err
	}

	source, err := iofs.New(database.Migrations, "migrations")
	if err != nil {
		return DB{}, err
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return DB{}, err
	}

	err = m.Up()
	if err != migrate.ErrNoChange && err != nil {
		return DB{}, err
	}

	return db, nil
}

func openDB(dsn string) (DB, error) {
	conn, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return DB{}, err
	}

	if err := conn.Ping(); err != nil {
		return DB{}, err
	}

	return DB{Conn: conn}, nil
}

type Session struct {
	ID        int        `db:"rowid"`
	Start     time.Time  `db:"start"`
	Stop      *time.Time `db:"stop"`
	Note      *string    `db:"note"`
	ProfileID int        `db:"profile_id"`
}

var ErrNoOpenSession = errors.New("no open session")

func (db DB) GetOpenSession(profileSlug string) (Session, error) {
	var session Session
	err := db.Conn.Get(&session, "SELECT s.`rowid`, s.`start`, s.`stop`, s.`profile_id` FROM `session` s INNER JOIN `profile` p ON s.`profile_id` = p.`id` WHERE s.`stop` IS NULL AND p.`slug` = ? LIMIT 1", profileSlug)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNoOpenSession
	} else if err != nil {
		return Session{}, err
	}

	return session, nil
}

func (db DB) StartSession(profileSlug string) error {
	_, err := db.Conn.Exec("INSERT INTO `session` (`start`, `profile_id`) VALUES (datetime('now'), (SELECT `id` FROM `profile` WHERE `slug` = ?))", profileSlug)
	if err != nil {
		return err
	}

	return nil
}

func (db DB) EndSession(s Session) error {
	res, err := db.Conn.Exec(
		"UPDATE `session` SET `stop` = datetime('now'), note = ? WHERE `rowid` = ?",
		s.Note,
		s.ID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if rows == 0 {
		return errors.New("session does not exist")
	} else if err != nil {
		return err
	}

	return nil
}
func (db DB) DeleteSession(id int) error {
	res, err := db.Conn.Exec("DELETE FROM `session` WHERE `rowid` = ?", id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if rows == 0 {
		return errors.New("session does not exist")
	} else if err != nil {
		return err
	}

	return nil
}

type listFilters struct {
    profileSlug string
	since  time.Time
	before time.Time
}

func (db DB) ListSessions(filters listFilters) ([]Session, error) {
	rows := []Session{}
	err := db.Conn.Select(
		&rows,
		"SELECT s.`rowid`, s.`start`, s.`stop`, s.`note`, s.`profile_id` FROM `session` s INNER JOIN `profile` p ON s.`profile_id` = p.`id` WHERE strftime('%s', s.`start`) BETWEEN strftime('%s', ?) AND strftime('%s', ?) AND p.`slug` = ? ORDER BY s.`rowid` ASC",
		filters.since,
		filters.before,
        filters.profileSlug,
	)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

type Profile struct {
	ID          int       `db:"id"`
	Slug        string    `db:"slug"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (db DB) ListProfiles() ([]Profile, error) {
	var profiles []Profile
	err := db.Conn.Select(
		&profiles,
		"SELECT `id`, `slug`, `description`, `created_at`, `updated_at` FROM `profile` ORDER BY `id`",
	)
	if err != nil {
		return nil, err
	}

	return profiles, nil
}

func (db DB) CreateProfile(p Profile) error {
    _, err := db.Conn.Exec("INSERT INTO `profile` (`slug`, `description`) VALUES (?, ?)", p.Slug, p.Description)
    if err != nil {
        return err
    }

    return nil
}
