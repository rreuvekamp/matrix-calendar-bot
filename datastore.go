package main

import (
	"fmt"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

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

func (s *dataStore) populateFromDB() error {
	cals, err := s.persist.fetchAllCalendars()
	if err != nil {
		return err
	}

	for _, uc := range cals {
		ud, err := s.userData(uc.UserID)
		if err != nil {
			return err
		}
		ud.calendars = append(ud.calendars, uc)
	}

	return nil
}

func (s *dataStore) userData(user id.UserID) (*userData, error) {
	d, _ := s.users[user]
	if d != nil {
		return d, nil
	}

	/*cals, err := s.persist.fetchCalendars(user)
	if err != nil {
		return nil, err
	}*/

	ud := userData{userID: user, persist: s.persist}

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

func (ud *userData) setupReminderTimer(cli *mautrix.Client) error {
	for _, uc := range ud.calendars {
		cal, err := uc.calendar()
		if err != nil {
			return err
		}
		evs, err := cal.events(time.Now(), time.Now().Add(24*time.Hour))
		if err != nil {
			return err
		}

		for _, event := range evs {
			ev := event
			go func() {
				fmt.Println("Setup reminder for:", ev.text)
				<-time.After(time.Until(ev.from))
				//sendMessage(cli, id.RoomID("!qvPycavGoabBgSxiDz:remi.im"), "Reminder for:"+ev.text)
				fmt.Println("Reminder for: " + ev.text)
			}()
		}

		<-time.After(100 * time.Millisecond)
	}

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
