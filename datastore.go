package main

import "maunium.net/go/mautrix/id"

type dataStore struct {
	//users map[id.UserID]*userData

	persist *sqlDB
}

func newDataStore(db *sqlDB) *dataStore {
	return &dataStore{db}
}

func (s *dataStore) userData(user id.UserID) (*userData, error) {
	//d, _ := s.users[user]

	cals, err := s.persist.fetchCalendars(user)
	if err != nil {
		return nil, err
	}

	return &userData{user, s.persist, cals}, nil
}

type userData struct {
	userID id.UserID

	persist *sqlDB

	calendars []userCalendar
}

func (ud *userData) addCalendar(uri string) error {
	_, err := ud.persist.addCalendar(ud.userID, uri)
	return err
}

type userCalendar struct {
	DBID int
	URI  string
}
