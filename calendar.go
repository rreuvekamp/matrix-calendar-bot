package main

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/dolanor/caldav-go/caldav"
	"github.com/dolanor/caldav-go/icalendar/components"
)

type calendar interface {
	events(from time.Time, until time.Time) (calendarEvents, error)
}

type calendarEvent struct {
	from, to time.Time

	text string
}

// calDavCalendar implements calendar
type calDavCalendar struct {
	mutex  sync.Mutex
	client *caldav.Client
}

func newCalDavCalendar(url string) (*calDavCalendar, error) {
	server, err := caldav.NewServer(url)
	if err != nil {
		return nil, err
	}

	client := caldav.NewDefaultClient(server)

	err = client.ValidateServer(url)
	return &calDavCalendar{client: client}, err
}

func (cal *calDavCalendar) events(from time.Time, until time.Time) (calendarEvents, error) {
	cal.mutex.Lock()
	calDavEvents, err := cal.client.GetEvents("")
	cal.mutex.Unlock()
	if err != nil {
		return []calendarEvent{}, err
	}

	return parseCaldavEvents(calDavEvents, from, until), nil
}

func parseCaldavEvents(evs []*components.Event, from, until time.Time) calendarEvents {
	events := []calendarEvent{}

	for _, ev := range evs {
		if from != (time.Time{}) && ev.DateStart.NativeTime().Before(from) {
			continue
		}

		if until != (time.Time{}) && until.Before(ev.DateStart.NativeTime()) {
			continue
		}

		event := calendarEvent{
			from: ev.DateStart.NativeTime(),
			to:   ev.DateEnd.NativeTime(),
			text: ev.Summary,
		}

		events = append(events, event)
	}

	sort.Sort(calendarEvents(events))

	return events
}

type iCalCalendar struct {
	client *caldav.Client
}

func newICalCalendar(url string) (*iCalCalendar, error) {
	server, err := caldav.NewServer(url)
	if err != nil {
		return nil, err
	}

	client := caldav.NewDefaultClient(server)

	cal := &iCalCalendar{client: client}

	_, err = cal.events(time.Now(), time.Now())

	return cal, err
}

func (cal *iCalCalendar) events(from time.Time, until time.Time) (calendarEvents, error) {
	req, err := cal.client.Server().NewRequest("GET", "")
	if err != nil {
		return nil, err
	}
	resp, err := cal.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot fetch ical file, status code: %d", resp.StatusCode)
	}

	calendar := components.Calendar{}
	err = resp.Decode(&calendar)
	if err != nil {
		return nil, err
	}

	return parseCaldavEvents(calendar.Events, from, until), nil
}

type combinedCalendar []calendar

var errNoCalendars = errors.New("no calendars")

func (cals combinedCalendar) events(from time.Time, until time.Time) (calendarEvents, error) {
	var events []calendarEvent

	if len(cals) == 0 {
		return events, errNoCalendars
	}

	for _, cal := range cals {
		evs, err := cal.events(from, until)
		if err != nil {
			// TODO: Multierror
			return events, err
		}

		events = append(events, []calendarEvent(evs)...)
	}

	sort.Sort(calendarEvents(events))

	return events, nil
}

type eventDay struct {
	dayStr string
	day    time.Time
	events []calendarEvent
}

func (evs calendarEvents) format() []*eventDay {
	days := []*eventDay{}
	for _, ev := range evs {

		cur := ev.from
		fromStr := ev.from.Format("2006-01-02")
		toStr := ev.to.Format("2006-01-02")
		for {
			curStr := cur.Format("2006-01-02")

			var thisDay *eventDay
			for i, day := range days {
				if day.dayStr == curStr {
					thisDay = days[i]
					break
				}
			}
			if thisDay == nil {
				thisDay = &eventDay{dayStr: curStr, day: cur}
				days = append(days, thisDay)
			}

			evCp := ev

			if fromStr != toStr {
				if fromStr != curStr {
					evCp.from = time.Date(evCp.from.Year(), evCp.from.Month(), evCp.from.Day(), 0, 0, 0, 0, evCp.from.Location())
				}
				if toStr != curStr {
					evCp.to = time.Date(evCp.to.Year(), evCp.to.Month(), evCp.to.Day(), 0, 0, 0, 0, evCp.to.Location())
				}
			}

			thisDay.events = append(thisDay.events, evCp)

			if cur.Format("2006-01-02") == ev.to.Format("2006-01-02") {
				break
			}

			cur = cur.AddDate(0, 0, 1)
		}
	}

	return days
}

type calendarType string

var (
	calendarTypeCalDav = calendarType("caldav")
	calendarTypeICal   = calendarType("ical")
)

// calendarEvents implements sort.Interface
type calendarEvents []calendarEvent

func (c calendarEvents) Len() int {
	return len(c)
}

func (c calendarEvents) Less(i, j int) bool {
	return c[i].from.Unix() < c[j].from.Unix()
}

func (c calendarEvents) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
