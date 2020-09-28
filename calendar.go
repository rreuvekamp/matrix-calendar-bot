package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/dolanor/caldav-go/caldav"
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
	client *caldav.Client
}

func newCalDavCalendar(url string) (*calDavCalendar, error) {
	server, err := caldav.NewServer(url)
	if err != nil {
		return nil, err
	}

	client := caldav.NewDefaultClient(server)

	err = client.ValidateServer(url)
	return &calDavCalendar{client}, err
}

func (cal *calDavCalendar) events(from time.Time, until time.Time) (calendarEvents, error) {
	calDavEvents, err := cal.client.GetEvents("")
	if err != nil {
		fmt.Println(err)
		return []calendarEvent{}, err
	}

	events := []calendarEvent{}

	for _, ev := range calDavEvents {
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

	return events, nil
}

type combinedCalendar []calendar

func (cals combinedCalendar) events(from time.Time, until time.Time) (calendarEvents, error) {
	var events []calendarEvent
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
