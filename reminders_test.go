package main

import (
	"fmt"
	"testing"
	"time"
)

func TestCreateRemindersCreatesTheCorrectReminders(t *testing.T) {
	ev0 := &calendarEvent{
		from: time.Date(2000, 10, 10, 10, 10, 10, 0, time.Local),
		to:   time.Date(2000, 10, 10, 11, 10, 10, 0, time.Local),
		text: "test event",
	}
	ev1 := &calendarEvent{
		from: time.Date(2010, 10, 10, 10, 10, 10, 0, time.Local),
		to:   time.Date(2010, 10, 10, 11, 10, 10, 0, time.Local),
		text: "test event 1",
	}

	ev2 := &calendarEvent{
		from: time.Now().Add(15 * time.Minute),
		to:   time.Now().Add(75 * time.Minute),
		text: "test event 2",
	}

	ev3 := &calendarEvent{
		from: time.Now().Add(29 * time.Minute),
		to:   time.Now().Add(75 * time.Minute),
		text: "test event 3",
	}

	ev4 := &calendarEvent{
		from: time.Now().Add(45 * time.Minute),
		to:   time.Now().Add(75 * time.Minute),
		text: "test event 4",
	}

	ev5 := &calendarEvent{
		from: time.Date(2150, 10, 10, 10, 10, 10, 0, time.Local),
		to:   time.Date(2150, 10, 10, 11, 10, 10, 0, time.Local),
		text: "test event 5",
	}

	events := []*calendarEvent{ev0, ev1, ev2, ev3, ev4, ev5}

	timer := newReminderTimer(nil, 30*time.Minute, newMockCalendar(events), []time.Duration{0 * time.Second, 30 * time.Minute})
	/*u := user{
		calendars: []*userCalendar{
			&userCalendar{
				cal: newMockCalendar(events),
			},
		},
	}*/

	reminders, err := timer.createReminders()
	if err != nil {
		t.Error(err)
	}

	for _, r := range reminders {
		t.Log(r.event.text)
	}

	if len(reminders) != 4 {
		t.Errorf("received incorrect amount of reminders, got: %d", len(reminders))
	}

	assertEqual(t, *reminders[0].event, *ev2, "reminder has correct event")
	assertEqual(t, *reminders[1].event, *ev4, "reminder has correct event")
	assertEqual(t, *reminders[2].event, *ev3, "reminder has correct event")
	assertEqual(t, *reminders[3].event, *ev4, "reminder has correct event")

	assertEqual(t, reminders[0].when, ev2.from, "reminder has correct when")
	assertEqual(t, reminders[1].when, ev4.from.Add(-30*time.Minute), "reminder has correct when")
	assertEqual(t, reminders[2].when, ev3.from, "reminder has correct when")
	assertEqual(t, reminders[3].when, ev4.from, "reminder has correct when")
}

func TestReminderLoopSendsTheCorrectReminders(t *testing.T) {
	ev0 := &calendarEvent{
		from: time.Now(),
		to:   time.Now().Add(75 * time.Minute),
		text: "test event 0",
	}
	r0 := reminder{
		time.Now(),
		ev0,
	}

	ev1 := &calendarEvent{
		from: time.Now(),
		to:   time.Now().Add(75 * time.Minute),
		text: "test event 1",
	}
	r1 := reminder{
		time.Now(),
		ev1,
	}
	ev2 := &calendarEvent{
		from: time.Now(),
		to:   time.Now().Add(75 * time.Minute),
		text: "test event 2",
	}
	r2 := reminder{
		time.Now(),
		ev2,
	}

	reminders := []reminder{r0, r1, r2}

	received := []*calendarEvent{}

	reminderCallback := func(ev *calendarEvent) {
		received = append(received, ev)
	}

	reminderLoop(reminders, nil, reminderCallback)

	if len(received) != 3 {
		t.Errorf("received incorrect amount of reminders, got: %d", len(received))
	}

	assertEqual(t, received[0], ev0, "reminder has correct event")
	assertEqual(t, received[1], ev1, "reminder has correct event")
	assertEqual(t, received[2], ev2, "reminder has correct event")
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("test if %s, %v != %v", message, a, b)
	t.Fatal(message)
}
