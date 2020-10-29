package main

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"maunium.net/go/mautrix/id"
)

type sqlDB struct {
	db *sql.DB

	stmtFetchCalendars    *sql.Stmt
	stmtFetchAllCalendars *sql.Stmt
	stmtAddCalendar       *sql.Stmt
	stmtRemoveCalendar    *sql.Stmt

	stmtFetchAllUsers    *sql.Stmt
	stmtAddUser          *sql.Stmt
	stmtUpdateUserRoomID *sql.Stmt
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

	d.stmtFetchCalendars, err = db.Prepare("SELECT id, user_id, name, cal_type, uri FROM calendar WHERE user_id = ?;")
	if err != nil {
		return d, err
	}

	d.stmtFetchAllCalendars, err = db.Prepare("SELECT id, user_id, name, cal_type, uri FROM calendar;")
	if err != nil {
		return d, err
	}

	d.stmtAddCalendar, err = db.Prepare("INSERT INTO calendar (user_id, name, cal_type, uri) VALUES (?, ?, ?, ?);")
	if err != nil {
		return d, err
	}

	d.stmtRemoveCalendar, err = db.Prepare("DELETE FROM calendar WHERE user_id = ? AND name = ?;")
	if err != nil {
		return d, err
	}

	d.stmtFetchAllUsers, err = db.Prepare("SELECT user_id, room_id FROM user;")
	if err != nil {
		return d, err
	}

	d.stmtAddUser, err = db.Prepare("INSERT INTO user (user_id, room_id) VALUES (?, ?);")
	if err != nil {
		return d, err
	}

	d.stmtUpdateUserRoomID, err = db.Prepare("UPDATE user SET room_id = ? WHERE user_id = ?;")
	return d, err
}

func (d *sqlDB) createTables() error {
	userSQL := `CREATE TABLE IF NOT EXISTS user (
		"user_id" TEXT NOT NULL PRIMARY KEY,
		"room_ID" TEXT,
		"created" datetime default current_timestamp);`
	_, err := d.db.Exec(userSQL)
	if err != nil {
		return err
	}

	// TODO: Make calendar have relation with user
	calendarSQL := `CREATE TABLE IF NOT EXISTS calendar (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"user_id" TEXT,
		"name" TEXT,
		"cal_type" TEXT,
		"uri" TEXT,
		"created" datetime default current_timestamp);`

	_, err = d.db.Exec(calendarSQL)
	return err
}

func (d *sqlDB) fetchAllUsers() ([]user, error) {
	rows, err := d.stmtFetchAllUsers.Query()
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	users := []user{}
	for rows.Next() {
		user := user{}
		var roomID string
		err = rows.Scan(&user.userID, &roomID)
		if err != nil {
			return users, err
		}
		user.roomID = id.RoomID(roomID)

		users = append(users, user)
	}

	return users, nil
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
		cal := userCalendar{mutex: &sync.RWMutex{}}
		var userID string
		var calTypeStr string
		err := rows.Scan(&cal.DBID, &userID, &cal.Name, &calTypeStr, &cal.URI)
		if err != nil {
			return cals, err
		}

		cal.UserID = id.UserID(userID)

		switch calTypeStr {
		case "caldav":
			cal.CalType = calendarTypeCalDav
		case "ical":
			cal.CalType = calendarTypeICal
		default:
			fmt.Printf("unknown caltype in database: %q, row id: %d\n", calTypeStr, cal.DBID)
			continue
		}

		cals = append(cals, cal)
	}

	return cals, nil
}

func (d *sqlDB) addCalendar(userID id.UserID, name string, calType calendarType, uri string) (int64, error) {
	res, err := d.stmtAddCalendar.Exec(userID, name, string(calType), uri)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	return id, err
}

func (d *sqlDB) removeCalendar(userID id.UserID, name string) error {
	_, err := d.stmtRemoveCalendar.Exec(userID, name)

	return err
}

func (d *sqlDB) updateUserRoomID(userID id.UserID, roomID id.RoomID) error {
	_, err := d.stmtUpdateUserRoomID.Exec(roomID, userID)

	return err
}

func (d *sqlDB) addUser(userID id.UserID, roomID id.RoomID) error {
	_, err := d.stmtAddUser.Exec(userID, roomID)

	return err
}
