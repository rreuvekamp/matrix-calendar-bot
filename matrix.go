package main

import (
	"fmt"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type matrixBot struct {
	cli *mautrix.Client
}

func initMatrixBot(cfg configMatrixBot, data *store) (matrixBot, error) {
	us := id.UserID(cfg.AccountID)
	cli, err := mautrix.NewClient(cfg.Homeserver, us, cfg.Token)
	m := matrixBot{cli}
	if err != nil {
		return m, err
	}

	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnSync(ignoreOldMessagesSyncHandler)
	syncer.OnEventType(event.EventMessage, func(_ mautrix.EventSource, ev *event.Event) {
		if ev.Sender == us {
			return
		}

		reply := handleCommand(cli, data, ev)
		for _, msg := range reply {
			m.sendNotice(ev.RoomID, msg.msg, msg.msgF)
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

	return m, nil
}

func (m matrixBot) sendNotice(roomID id.RoomID, msg string, msgF string) error {
	return m.sendMatrixMessage(roomID, msg, msgF, event.MsgNotice)
}

func (m matrixBot) sendMessage(roomID id.RoomID, msg string, msgF string) error {
	return m.sendMatrixMessage(roomID, msg, msgF, event.MsgText)
}

func (m matrixBot) sendMatrixMessage(roomID id.RoomID, msg string, msgF string, eventType event.MessageType) error {
	ev := event.MessageEventContent{
		MsgType: eventType,
		Body:    msg,
	}
	if msgF != "" {
		ev.FormattedBody = msgF
		ev.Format = event.FormatHTML
	}
	_, err := m.cli.SendMessageEvent(roomID, event.EventMessage, ev)
	return err
}

func ignoreOldMessagesSyncHandler(resp *mautrix.RespSync, since string) bool {
	return since != ""
}
