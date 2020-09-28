package main

import (
	"fmt"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

// store is in-memory store of bot users.
type store struct {
	users map[id.UserID]*user

	persist *sqlDB
}

func newDataStore(db *sqlDB) *store {
	return &store{
		users:   make(map[id.UserID]*user),
		persist: db,
	}
}

func (s *store) populateFromDB() error {
	cals, err := s.persist.fetchAllCalendars()
	if err != nil {
		return err
	}

	for _, uc := range cals {
		u, err := s.user(uc.UserID)
		if err != nil {
			return err
		}
		u.calendars = append(u.calendars, uc)
	}

	return nil
}

func (s *store) user(id id.UserID) (*user, error) {
	d, _ := s.users[id]
	if d != nil {
		return d, nil
	}

	u := user{userID: id, persist: s.persist}

	s.users[id] = &u

	return &u, nil
}

type user struct {
	userID id.UserID

	persist *sqlDB

	calendars []userCalendar
}

func (u *user) addCalendar(uri string) error {
	dbid, err := u.persist.addCalendar(u.userID, uri)
	if err != nil {
		return err
	}

	uc := userCalendar{DBID: dbid, URI: uri}
	u.calendars = append(u.calendars, uc)

	return nil
}

func (u *user) setupReminderTimer(cli *mautrix.Client) error {
	for _, uc := range u.calendars {
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
