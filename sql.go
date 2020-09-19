package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"maunium.net/go/mautrix/id"
)

type sqlDB struct {
	db *sql.DB

	stmtFetchCalendars *sql.Stmt
	stmtAddCalendar    *sql.Stmt
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

	d.stmtFetchCalendars, err = db.Prepare("SELECT id, uri FROM calendar WHERE user_id = ?;")
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
		"uri" TEXT );`
	// TODO: updated, created

	_, err := d.db.Exec(calendarSQL)
	return err
}

func (d *sqlDB) fetchCalendars(userID id.UserID) ([]userCalendar, error) {
	rows, err := d.stmtFetchCalendars.Query(userID)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	cals := []userCalendar{}
	for rows.Next() {
		cal := userCalendar{}
		err = rows.Scan(&cal.DBID, &cal.URI)
		if err != nil {
			return cals, err
		}

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
