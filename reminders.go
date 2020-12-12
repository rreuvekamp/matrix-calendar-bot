package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type reminderTimer struct {
	send func(*calendarEvent)

	forDuration time.Duration

	reminderTimes []time.Duration

	cal queryableCalendar

	stopTimer      chan struct{}
	stopTimerMutex sync.Mutex
}

func newReminderTimer(send func(*calendarEvent), forDuration time.Duration, cal queryableCalendar, reminderTimes []time.Duration) reminderTimer {
	return reminderTimer{
		send:          send,
		forDuration:   forDuration,
		reminderTimes: reminderTimes,
		cal:           cal,
	}
}

func (t *reminderTimer) set() error {
	reminders, err := t.createReminders()
	if err != nil {
		return err
	}

	t.stopTimerMutex.Lock()
	defer t.stopTimerMutex.Unlock()

	if t.stopTimer != nil {
		t.stopTimer <- struct{}{}
	}

	t.stopTimer = make(chan struct{}, 1)

	go reminderLoop(reminders, t.stopTimer, t.send)

	return nil
}

func (t *reminderTimer) createReminders() ([]reminder, error) {
	evs, err := t.cal.eventsBetween(time.Now(), time.Now().Add(t.forDuration).Add(t.highestReminderTime()))
	if err != nil {
		return []reminder{}, err
	}

	rems := []reminder{}

	for _, ev := range evs {
		for _, remT := range t.reminderTimes {
			remTime := ev.from.Add(-remT)

			if time.Now().Before(remTime) {
				remPre := reminder{
					when:  remTime,
					event: ev,
				}

				rems = append(rems, remPre)
			}
		}
	}

	sort.Sort(reminders(rems))

	return rems, nil
}

func reminderLoop(reminders []reminder, stop <-chan struct{}, send func(*calendarEvent)) {
	for {
		if len(reminders) == 0 {
			break
		}

		next := reminders[0]

		fmt.Println("Setup reminder for: " + next.event.text)

		select {
		case <-stop:
			fmt.Println("Reminderloop stopped")
			break
		case <-time.After(time.Until(next.when)):
			send(next.event)
			fmt.Println("Reminder for:", next.event.text, next.event.from.Sub(time.Now()))

			reminders = append([]reminder{}, reminders[1:]...)
		}
	}
}

func (t reminderTimer) highestReminderTime() time.Duration {
	highest := 0 * time.Second
	for _, remT := range t.reminderTimes {
		if remT > highest {
			highest = remT
		}
	}

	return highest
}

type reminder struct {
	when time.Time

	event *calendarEvent
}

type reminders []reminder

func (r reminders) Len() int {
	return len(r)
}

func (r reminders) Less(i, j int) bool {
	return r[i].when.Unix() < r[j].when.Unix()
}

func (r reminders) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
