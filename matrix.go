package main

import (
	"fmt"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func initMatrixBot(cfg configMatrixBot, data *store) (*mautrix.Client, error) {
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

		reply := handleCommand(cli, data, ev)
		for _, msg := range reply {
			sendNotice(cli, ev.RoomID, msg.msg, msg.msgF)
		}
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

func sendNotice(cli *mautrix.Client, roomID id.RoomID, msg string, msgF string) error {
	return sendMatrixMessage(cli, roomID, msg, msgF, event.MsgNotice)
}

func sendMessage(cli *mautrix.Client, roomID id.RoomID, msg string, msgF string) error {
	return sendMatrixMessage(cli, roomID, msg, msgF, event.MsgText)
}

func sendMatrixMessage(cli *mautrix.Client, roomID id.RoomID, msg string, msgF string, eventType event.MessageType) error {
	ev := event.MessageEventContent{
		MsgType: eventType,
		Body:    msg,
	}
	if msgF != "" {
		ev.FormattedBody = msgF
		ev.Format = event.FormatHTML
	}
	_, err := cli.SendMessageEvent(roomID, event.EventMessage, ev)
	return err
}

func ignoreOldMessagesSyncHandler(resp *mautrix.RespSync, since string) bool {
	return since != ""
}
