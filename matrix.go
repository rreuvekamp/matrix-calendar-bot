package main

import (
	"fmt"
	"strings"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func initMatrixBot(cfg configMatrixBot, ds *dataStore) (*mautrix.Client, error) {
	us := id.UserID(cfg.AccountID)
	cli, err := mautrix.NewClient(cfg.Homeserver, us, cfg.Token)
	if err != nil {
		return nil, err
	}

	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnSync(ignoreOldMessagesSyncHandler)
	syncer.OnEventType(event.EventMessage, func(_ mautrix.EventSource, ev *event.Event) {
		if ev.Sender == us {
			return
		}

		handleMessage(cli, ds, ev)
	})
	syncer.OnEventType(event.StateMember, func(_ mautrix.EventSource, ev *event.Event) {
		if ev.Sender == us {
			return
		}
		if ev.Content.AsMember().Membership != "invite" {
			return
		}

		fmt.Println("Invite: ", ev)
		// TODO: Welcome message
		// TODO: Support only 1-1 rooms

		resp, err := cli.JoinRoom(ev.RoomID.String(), "", nil)
		fmt.Println("JoinRoom response:", resp)
		if err != nil {
			fmt.Println(err)
		}
	})

	// Non-blocking version
	go func() {
		backOff := 0
		for {
			if err := cli.Sync(); err != nil {
				fmt.Println("Sync() returned ", err)
				sleep := backOff * 2
				<-time.After(time.Duration(sleep) * time.Second)
				backOff++
				continue
			}
			backOff = 0
		}
	}()

	return cli, nil
}

func handleMessage(cli *mautrix.Client, ds *dataStore, ev *event.Event) {
	var err error

	str := strings.TrimSpace(ev.Content.AsMessage().Body)
	str = strings.ToLower(str)

	args := strings.Split(str, " ")
	if len(args) < 0 {
		return
	}

	ud, err := ds.userData(ev.Sender)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch args[0] {
	case "listevents":
		err = cmdListEvents(cli, ud, ev.RoomID)
	case "addcalendar":
		err = cmdAddCalendar(cli, ud, ev.RoomID, args)
	default:
		err = sendMessage(cli, ev.RoomID, "Unknown command")
	}

	if err != nil {
		fmt.Println(err)
	}
}

func cmdListEvents(cli *mautrix.Client, ud *userData, roomID id.RoomID) error {
	cal, err := ud.calendars[0].calendar()
	if err != nil {
		return err
	}

	now := time.Now()
	startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-int(now.Weekday())+1, 0, 0, 0, 0, now.Location())
	endOfWeek := startOfWeek
	endOfWeek = endOfWeek.Add(7 * 24 * time.Hour)

	events, err := cal.events(startOfWeek, endOfWeek)
	if err != nil {
		return err
	}

	// TODO: Properly handle multi-day events.

	msg := ""
	last := time.Time{}
	for _, calEv := range events {
		if last == (time.Time{}) || calEv.from.Format("2006-01-02") != last.Format("2006-01-02") {
			// Different day from last event

			msg += fmt.Sprintf("\n%s\n", calEv.from.Format("Monday 2 Januari"))
		}

		msg += fmt.Sprintf("%s - %s: %s\n", calEv.from.Format("15:04"), calEv.to.Format("15:04"), calEv.text)

		last = calEv.from
	}

	return sendMessage(cli, roomID, msg)
}

func cmdAddCalendar(cli *mautrix.Client, ud *userData, roomID id.RoomID, args []string) error {
	if len(args) < 2 {
		sendMessage(cli, roomID, "Provide the URI")
		// TODO: Improve message
		return nil
	}

	uri := args[1]

	_, err := newCalDavCalendar(uri)
	if err != nil {
		sendMessage(cli, roomID, "Specified URI is not a supported CalDAV calendar")
		fmt.Println(err)
		return nil
	}

	return ud.addCalendar(uri)
}

func sendMessage(cli *mautrix.Client, roomID id.RoomID, msg string) error {
	_, err := cli.SendMessageEvent(roomID, event.EventMessage, event.MessageEventContent{
		Body: msg,
	})
	return err
}

func ignoreOldMessagesSyncHandler(resp *mautrix.RespSync, since string) bool {
	return since != ""
}
