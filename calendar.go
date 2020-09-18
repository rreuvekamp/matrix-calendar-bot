package main

import (
	"fmt"
	"time"

	"github.com/dolanor/caldav-go/caldav"
)

type calendar interface {
	events() []calendarEvent
}

type calendarEvent struct {
	from, to time.Time

	text string
}

type calDavCalendar struct {
	client *caldav.Client
}

func newCalDavCalendar(url string) (*calDavCalendar, error) {
	server, err := caldav.NewServer(url)
	if err != nil {
		return nil, err
	}

	client := caldav.NewDefaultClient(server)

	// start executing requests!
	err = client.ValidateServer(url)
	return &calDavCalendar{client}, err
}

func (cal *calDavCalendar) events() ([]calendarEvent, error) {
	calDavEvents, err := cal.client.GetEvents("")
	if err != nil {
		fmt.Println(err)
		return []calendarEvent{}, err
	}

	events := []calendarEvent{}

	now := time.Now()
	for _, ev := range calDavEvents {
		if ev.DateStart.NativeTime().Before(now) {
			continue
		}

		event := calendarEvent{
			from: ev.DateStart.NativeTime(),
			to:   ev.DateEnd.NativeTime(),
			text: ev.Summary,
		}

		events = append(events, event)
	}

	return events, nil
}
