package main

import (
	"errors"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/apognu/gocal"
	"github.com/dolanor/caldav-go/caldav"
)

// calendar allows fetching events.
type calendar interface {
	events() (calendarEvents, error)
}

// calendarEvent represents a single calendar item.
type calendarEvent struct {
	from, to time.Time

	text string
}

// cachedCalendar wraps a calendar caching its events.
type cachedCalendar struct {
	cal    calendar
	period time.Duration

	cache       calendarEvents
	lastUpdated time.Time
	cleanTimer  *time.Timer
	mutex       sync.RWMutex
}

// newCachedCalendar wrapping the given calendar, caching its events for the given period.
func newCachedCalendar(cal calendar, period time.Duration) *cachedCalendar {
	return &cachedCalendar{cal: cal, period: period}
}

func (cal *cachedCalendar) events() (calendarEvents, error) {
	cal.mutex.RLock()

	if cal.cache == nil {
		cal.mutex.RUnlock()

		cal.mutex.Lock()
		defer cal.mutex.Unlock()

		var err error
		cal.cache, err = cal.cal.events()
		if err != nil {
			return cal.cache, err
		}

		cal.lastUpdated = time.Now()

		if cal.cleanTimer != nil {
			cal.cleanTimer.Stop()
		}

		cal.cleanTimer = time.AfterFunc(cal.period, cal.clean)

		return cal.cache, err
	}

	defer cal.mutex.RUnlock()

	return cal.cache, nil
}

func (cal *cachedCalendar) clean() {
	cal.mutex.Lock()
	defer cal.mutex.Unlock()

	cal.cache = nil
	cal.cleanTimer = nil
}

// calDavCalendar implements calendar, fetches events from a caldav server.
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

func (cal *calDavCalendar) events() (calendarEvents, error) {
	cal.mutex.Lock()
	evs, err := cal.client.GetEvents("")
	cal.mutex.Unlock()
	if err != nil {
		return []*calendarEvent{}, err
	}

	events := []*calendarEvent{}

	for _, ev := range evs {
		event := calendarEvent{
			from: ev.DateStart.NativeTime(),
			to:   ev.DateEnd.NativeTime(),
			text: ev.Summary,
		}

		events = append(events, &event)
	}

	sort.Sort(calendarEvents(events))

	return events, nil
}

// iCalCalendar implements calendar, fetches events from a remote ical file.
type iCalCalendar struct {
	url string
}

func newICalCalendar(url string) (*iCalCalendar, error) {
	return &iCalCalendar{url}, nil
}

func (cal *iCalCalendar) events() (calendarEvents, error) {
	// TODO: user agent
	resp, err := http.Get(cal.url)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	c := gocal.NewParser(resp.Body)
	c.Parse()

	events := []*calendarEvent{}

	for _, ev := range c.Events {
		var start, end time.Time
		if ev.Start != nil {
			start = *ev.Start
		}
		if ev.End != nil {
			end = *ev.End
		}

		event := calendarEvent{
			from: start,
			to:   end,
			text: ev.Summary,
		}

		events = append(events, &event)
	}

	sort.Sort(calendarEvents(events))

	return events, nil
}

// combinedCalendar wraps multipe calendars.
type combinedCalendar []calendar

var errNoCalendars = errors.New("no calendars")

// events gives the calendarEvents from the underlying calendars which start between
// the gives dates.
func (cals combinedCalendar) events(from time.Time, until time.Time) (calendarEvents, error) {
	var events []*calendarEvent

	if len(cals) == 0 {
		return events, errNoCalendars
	}

	for _, cal := range cals {
		evs, err := cal.events()
		evs = evs.between(from, until)
		if err != nil {
			// TODO: Multierror
			return events, err
		}

		events = append(events, []*calendarEvent(evs)...)
	}

	sort.Sort(calendarEvents(events))

	return events, nil
}

type eventDay struct {
	dayStr string
	day    time.Time
	events []calendarEvent
}

type calendarType string

var (
	calendarTypeCalDav = calendarType("caldav")
	calendarTypeICal   = calendarType("ical")
)

// calendarEvents implements sort.Interface
type calendarEvents []*calendarEvent

func (c calendarEvents) between(from time.Time, until time.Time) calendarEvents {
	var events []*calendarEvent

	for _, ev := range c {
		if from != (time.Time{}) && ev.from.Before(from) {
			continue
		}

		if until != (time.Time{}) && until.Before(ev.from) {
			continue
		}

		events = append(events, ev)
	}

	return events
}

// formatsToDays converts the events into days, to ease printing a calendar.
func (evs calendarEvents) formatToDays() []*eventDay {
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

			evCp := *ev

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

func (c calendarEvents) Len() int {
	return len(c)
}

func (c calendarEvents) Less(i, j int) bool {
	return c[i].from.Unix() < c[j].from.Unix()
}

func (c calendarEvents) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
