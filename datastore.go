package main

import (
	"fmt"
	"sync"
	"time"

	"maunium.net/go/mautrix/id"
)

// store is in-memory store of bot users.
type store struct {
	usersMutex sync.RWMutex
	users      map[id.UserID]*user

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
	s.usersMutex.RLock()
	d, _ := s.users[id]
	s.usersMutex.RUnlock()

	if d != nil {
		return d, nil
	}

	u := user{userID: id, persist: s.persist}

	s.usersMutex.Lock()
	s.users[id] = &u
	s.usersMutex.Unlock()

	return &u, nil
}

type user struct {
	mutex     sync.RWMutex
	userID    id.UserID
	persist   *sqlDB
	timerQuit chan struct{}

	calendarsMutex sync.RWMutex
	calendars      []userCalendar
}

func (u *user) addCalendar(uri string) error {
	u.mutex.RLock()
	id := u.userID
	u.mutex.RUnlock()

	dbid, err := u.persist.addCalendar(id, uri)
	if err != nil {
		return err
	}

	uc := userCalendar{DBID: dbid, URI: uri}

	u.mutex.Lock()
	u.calendars = append(u.calendars, uc)
	u.mutex.RUnlock()

	return nil
}

func (u *user) combinedCalendar() (calendar, error) {
	u.calendarsMutex.RLock()
	defer u.calendarsMutex.RUnlock()

	cals := make([]calendar, 0, len(u.calendars))

	for _, uc := range u.calendars {
		cal, err := uc.calendar()
		if err != nil {
			return combinedCalendar(cals), err
		}

		cals = append(cals, cal)
	}

	return combinedCalendar(cals), nil
}

func (u *user) setupReminderTimer(send func(calendarEvent)) error {
	u.calendarsMutex.RLock()
	cal, err := u.combinedCalendar()
	u.calendarsMutex.RUnlock()

	if err != nil {
		return err
	}

	evs, err := cal.events(time.Now(), time.Now().Add(5*time.Hour))
	if err != nil {
		return err
	}

	u.mutex.RLock()
	if u.timerQuit != nil {
		u.timerQuit <- struct{}{}
	}
	u.mutex.RUnlock()

	quit := make(chan struct{})

	u.mutex.Lock()
	u.timerQuit = quit
	u.mutex.Unlock()

	go func() {
		for _, ev := range evs {
			fmt.Println("Setup reminder for:", ev.text)

			select {
			case <-quit:
				return
			case <-time.After(time.Until(ev.from)):
			}
			send(ev)
			fmt.Println("Reminder for: " + ev.text)
		}
	}()

	return nil
}

type userCalendar struct {
	mutex sync.RWMutex

	DBID   int64
	UserID id.UserID
	URI    string

	cal calendar
}

func (uc *userCalendar) calendar() (calendar, error) {
	uc.mutex.Lock()
	defer uc.mutex.Unlock()

	var err error
	if uc.cal == nil {
		uc.cal, err = newCalDavCalendar(uc.URI)
	}

	return uc.cal, err
}
