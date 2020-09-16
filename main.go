package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dolanor/caldav-go/caldav"
	"github.com/dolanor/caldav-go/icalendar/components"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func main() {
	us := id.UserID("@calendartest:remi.im")
	cli, err := mautrix.NewClient("https://remi.im", us, os.Getenv("TOKEN"))
	if err != nil {
		fmt.Println(err)
		return
	}

	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnSync(IgnoreOldMessagesSyncHandler)
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

	fmt.Println("Started")

	<-make(chan struct{})
}

func handleMessage(cli *mautrix.Client, ev *event.Event) {
	calEvs := initCalendar()

	for _, calEv := range calEvs {
		fmt.Println("Message: ", ev.Content.AsMessage().Body)
		resp, err := cli.SendMessageEvent(ev.RoomID, event.EventMessage, event.MessageEventContent{
			Body: calEv.Summary,
		})
		fmt.Println(resp)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func initCalendar() (futureEvents []*components.Event) {
	server, err := caldav.NewServer(os.Getenv("CAL"))
	if err != nil {
		fmt.Println(err)
		return nil
	}

	client := caldav.NewDefaultClient(server)

	// start executing requests!
	err = client.ValidateServer(os.Getenv("CAL"))
	if err != nil {
		fmt.Println(err)
		return
	}

	events, err := client.GetEvents("")
	if err != nil {
		fmt.Println(err)
		return
	}

	now := time.Now()
	for _, ev := range events {
		if ev.DateStart.NativeTime().Before(now) {
			continue
		}

		futureEvents = append(futureEvents, ev)
		fmt.Println(ev)
	}

	return futureEvents
}

func IgnoreOldMessagesSyncHandler(resp *mautrix.RespSync, since string) bool {
	return since != ""
}
