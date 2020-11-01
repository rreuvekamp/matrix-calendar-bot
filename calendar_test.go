package main

import (
	"testing"
	"time"
)

func TestCachedCalendarEventsThenCacheIsPopulated(t *testing.T) {
	cc := newCachedCalendar(emptyCalendar{}, 5*time.Second)
	cc.events()

	if cc.cache == nil {
		t.Error("cache was not populated")
	}
}

func TestCachedCalendarCleanThenCacheIsCleaned(t *testing.T) {
	cc := newCachedCalendar(emptyCalendar{}, 5*time.Second)
	cc.events()

	cc.clean()

	if cc.cache != nil {
		t.Error("cache was not cleaned")
	}
}

func TestCachedCalendarEventsThenTimerIsSet(t *testing.T) {
	cc := newCachedCalendar(emptyCalendar{}, 5*time.Second)
	cc.events()

	if cc.cleanTimer == nil {
		t.Error("timer was not set")
	}
}

func TestCachedCalendarCleanThenTimerIsRemoved(t *testing.T) {
	cc := newCachedCalendar(emptyCalendar{}, 5*time.Second)
	cc.events()

	cc.clean()

	if cc.cleanTimer != nil {
		t.Error("timer was not removed")
	}
}

type emptyCalendar struct{}

func (c emptyCalendar) events() (calendarEvents, error) {
	return calendarEvents([]*calendarEvent{}), nil
}
