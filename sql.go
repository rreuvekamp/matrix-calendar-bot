package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"maunium.net/go/mautrix/id"
)

type sqlDB struct {
	db *sql.DB

	stmtFetchCalendars    *sql.Stmt
	stmtFetchAllCalendars *sql.Stmt
	stmtAddCalendar       *sql.Stmt
}

func initSQLDB(path string) (*sqlDB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	d := &sqlDB{db: db}

	err = d.createTables()
	if err != nil {
		return d, err
	}

	d.stmtFetchCalendars, err = db.Prepare("SELECT id, user_id, uri FROM calendar WHERE user_id = ?;")
	if err != nil {
		return d, err
	}

	d.stmtFetchAllCalendars, err = db.Prepare("SELECT id, user_id, uri FROM calendar;")
	if err != nil {
		return d, err
	}

	d.stmtAddCalendar, err = db.Prepare("INSERT INTO calendar (user_id, uri) VALUES (?, ?);")
	return d, err
}

func (d *sqlDB) createTables() error {
	calendarSQL := `CREATE TABLE IF NOT EXISTS calendar (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"user_id" TEXT,
		"uri" TEXT,
		"created" datetime default current_timestamp);`

	_, err := d.db.Exec(calendarSQL)
	return err
}

func (d *sqlDB) fetchAllCalendars() ([]userCalendar, error) {
	rows, err := d.stmtFetchAllCalendars.Query()
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	return rowsToCalendars(rows)
}

func (d *sqlDB) fetchCalendars(userID id.UserID) ([]userCalendar, error) {
	rows, err := d.stmtFetchCalendars.Query(userID)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	return rowsToCalendars(rows)
}

func rowsToCalendars(rows *sql.Rows) ([]userCalendar, error) {
	cals := []userCalendar{}
	for rows.Next() {
		cal := userCalendar{}
		var userID string
		err := rows.Scan(&cal.DBID, &userID, &cal.URI)
		if err != nil {
			return cals, err
		}

		cal.UserID = id.UserID(userID)

		cals = append(cals, cal)
	}

	return cals, nil
}

func (d *sqlDB) addCalendar(userID id.UserID, uri string) (int64, error) {
	res, err := d.stmtAddCalendar.Exec(userID, uri)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	return id, err
}
