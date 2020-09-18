package main

import (
	"fmt"
	"os"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func initMatrixBot() (*mautrix.Client, error) {
	us := id.UserID("@calendartest:remi.im")
	cli, err := mautrix.NewClient("https://remi.im", us, os.Getenv("TOKEN"))
	if err != nil {
		return nil, err
	}

	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnSync(ignoreOldMessagesSyncHandler)
	syncer.OnEventType(event.EventMessage, func(_ mautrix.EventSource, ev *event.Event) {
		if ev.Sender == us {
			return
		}

		handleMessage(cli, ev)
	})
	syncer.OnEventType(event.StateMember, func(_ mautrix.EventSource, ev *event.Event) {
		if ev.Sender == us {
			return
		}
		if ev.Content.AsMember().Membership != "invite" {
			return
		}

		fmt.Println("Invite: ", ev)

		resp, err := cli.JoinRoom(ev.RoomID.String(), "", nil)
		fmt.Println("JoinRoom response:", resp)
		if err != nil {
			fmt.Println(err)
		}
	})

	// Non-blocking version
	go func() {
		for {
			if err := cli.Sync(); err != nil {
				fmt.Println("Sync() returned ", err)
			}
			// Optional: Wait a period of time before trying to sync again.
		}
	}()

	return cli, nil
}

func handleMessage(cli *mautrix.Client, ev *event.Event) {
	var err error
	switch strings.TrimSpace(ev.Content.AsMessage().Body) {
	case "listevents":
		err = cmdListEvents(cli, ev.RoomID)
	default:
		err = sendMessage(cli, ev.RoomID, "Unknown command")
	}

	if err != nil {
		fmt.Println(err)
	}
}

func cmdListEvents(cli *mautrix.Client, roomID id.RoomID) error {
	cal, err := newCalDavCalendar(os.Getenv("CAL"))
	if err != nil {
		return err
	}

	events, err := cal.events()
	if err != nil {
		return err
	}

	msg := ""
	for _, calEv := range events {
		msg += fmt.Sprintf("%s - %s: %s\n", calEv.from.Format("2006-01-02 15:04"), calEv.to.Format("15:04"), calEv.text)
	}

	return sendMessage(cli, roomID, msg)
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
