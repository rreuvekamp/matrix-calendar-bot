package main

import "maunium.net/go/mautrix/id"

type dataStore struct {
	users map[id.UserID]*userData

	persist *sqlDB
}

func newDataStore(db *sqlDB) *dataStore {
	return &dataStore{
		users:   make(map[id.UserID]*userData),
		persist: db,
	}
}

func (s *dataStore) userData(user id.UserID) (*userData, error) {
	d, _ := s.users[user]
	if d != nil {
		return d, nil
	}

	cals, err := s.persist.fetchCalendars(user)
	if err != nil {
		return nil, err
	}

	ud := userData{user, s.persist, cals}

	s.users[user] = &ud

	return &ud, nil
}

type userData struct {
	userID id.UserID

	persist *sqlDB

	calendars []userCalendar
}

func (ud *userData) addCalendar(uri string) error {
	dbid, err := ud.persist.addCalendar(ud.userID, uri)
	if err != nil {
		return err
	}

	uc := userCalendar{DBID: dbid, URI: uri}
	ud.calendars = append(ud.calendars, uc)

	return nil
}

type userCalendar struct {
	DBID   int64
	UserID id.UserID
	URI    string

	cal calendar
}

func (uc *userCalendar) calendar() (calendar, error) {
	var err error
	if uc.cal == nil {
		uc.cal, err = newCalDavCalendar(uc.URI)
	}

	return uc.cal, err
}
