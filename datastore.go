package main

import "maunium.net/go/mautrix/id"

type dataStore struct {
	//users map[id.UserID]*userData
}

func newDataStore() *dataStore {
	return &dataStore{}
}

func (s *dataStore) userData(user id.UserID) *userData {
	//d, _ := s.users[user]
	return nil
}

type userData struct {
	calendars []userCalendar
}

type userCalendar struct {
	DBID int
	URI  string
}
