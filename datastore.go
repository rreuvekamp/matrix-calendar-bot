package main

import (
	"errors"
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
	users, err := s.persist.fetchAllUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		user.persist = s.persist
		user.existsInDB = true

		s.usersMutex.Lock()
		s.users[user.userID] = user
		s.usersMutex.Unlock()
	}

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
	userID     id.UserID
	roomID     id.RoomID
	existsInDB bool
	mutex      sync.RWMutex

	calendarsMutex sync.RWMutex
	calendars      []*userCalendar

	persist *sqlDB

	reminders             []reminder
	reminderNotifyUpdated chan struct{}
	remindersMutex        sync.RWMutex
}

func (u *user) store(roomID id.RoomID) error {
	u.mutex.RLock()
	userID := u.userID
	u.mutex.RUnlock()

	err := u.persist.addUser(userID, roomID)
	if err != nil {
		return err
	}

	u.mutex.Lock()
	u.roomID = roomID
	u.existsInDB = true
	u.mutex.Unlock()

	return nil
}

func (u *user) storeRoomID(roomID id.RoomID) error {
	u.mutex.RLock()
	userID := u.userID
	u.mutex.RUnlock()

	err := u.persist.updateUserRoomID(userID, roomID)
	if err != nil {
		return err
	}

	u.mutex.Lock()
	u.roomID = roomID
	u.mutex.Unlock()

	return nil
}

func (u *user) addCalendar(name string, calType calendarType, uri string) error {
	u.mutex.RLock()
	userID := u.userID
	u.mutex.RUnlock()

	dbid, err := u.persist.addCalendar(userID, name, calType, uri)
	if err != nil {
		return err
	}

	uc := userCalendar{DBID: dbid, Name: name, CalType: calType, URI: uri}

	u.mutex.Lock()
	u.calendars = append(u.calendars, &uc)
	u.mutex.Unlock()

	return nil
}

var errCalendarNotExists = errors.New("calendar doesn't exist")

func (u *user) removeCalendar(name string) error {
	found := 0

	u.calendarsMutex.RLock()
	userID := u.userID
	for i, cal := range u.calendars {
		if cal.Name != name {
			continue
		}

		found = i
		break
	}
	u.calendarsMutex.RUnlock()

	if found == 0 {
		return errCalendarNotExists
	}

	u.calendarsMutex.Lock()
	u.calendars = append(u.calendars[:found], u.calendars[found+1:]...)
	u.calendarsMutex.Unlock()

	return u.persist.removeCalendar(userID, name)
}

func (u *user) combinedCalendar() (combinedCalendar, error) {
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

func (u *user) hasCalendar(name string) bool {
	u.calendarsMutex.RLock()
	defer u.calendarsMutex.RUnlock()
	for _, cal := range u.calendars {
		if cal.Name != name {
			continue
		}

		return true
	}

	return false
}

func (u *user) setupReminderTimer(send func(*calendarEvent, id.RoomID), until time.Time) error {
	u.calendarsMutex.RLock()
	cal, err := u.combinedCalendar()
	u.calendarsMutex.RUnlock()

	// TODO: @@@@@@@@@@@@
	// Rewrote timers, haven't tested it yet.
	// Go fix it.

	if err != nil {
		return err
	}

	evs, err := cal.events(time.Now(), until)
	if err != nil {
		return err
	}

	newReminders := []reminder{}

	for _, ev := range evs {
		rem := reminder{
			when:  ev.from,
			event: ev,
		}

		newReminders = append(newReminders, rem)
	}

	u.remindersMutex.Lock()

	mustUpdate := false
	if len(u.reminders) > 0 && u.reminders[0] != newReminders[0].event {
		mustUpdate = true
	}

	u.reminders = newReminders

	if u.reminderNotifyUpdated == nil {
		u.reminderNotifyUpdated = make(chan struct{})
		go u.reminderLoop()
	} else if mustUpdate {
		u.reminderNotifyUpdated <- struct{}{}
	}
	u.reminderMutex.Unlock()

	return nil
}

func (u *user) reminderLoop() {
	for {
		u.remindersMutex.RLock()
		if len(u.reminders) == 0 {
			u.remindersMutex.RUnlock()
			break
		}

		rem := u.reminders[0]

		u.remindersMutex.RLock()

		fmt.Println("Setup reminder for:", rem)

		select {
		case <-quit:
			return
		case <-time.After(time.Until(rem.when)):
		}

		u.remindersMutex.Lock()
		u.reminders = append(u.reminders[1:]...)
		u.remindersMutex.Unlock()

		send(ev, u.RoomID())
		fmt.Println("Reminder for: " + ev.text)
	}
}

type reminder struct {
	when time.Time

	event *calendarEvent
}

func (u *user) ExistsInDB() bool {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.existsInDB
}

func (u *user) RoomID() id.RoomID {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.roomID
}

type userCalendar struct {
	mutex sync.RWMutex

	DBID    int64
	UserID  id.UserID
	Name    string
	CalType calendarType
	URI     string

	cal calendar
}

func (uc *userCalendar) calendar() (calendar, error) {
	uc.mutex.Lock()
	defer uc.mutex.Unlock()

	var err error
	if uc.cal == nil {
		switch uc.CalType {
		case calendarTypeICal:
			uc.cal, err = newICalCalendar(uc.URI)
		case calendarTypeCalDav:
			uc.cal, err = newCalDavCalendar(uc.URI)
		}

		// TODO: Cache time from config.
		uc.cal = newCachedCalendar(uc.cal, 5*time.Minute)
	}

	return uc.cal, err
}
