package main

import (
	"fmt"
	"strings"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type cmdReply struct {
	msg  string
	msgF string
}

func handleCommand(cli *mautrix.Client, data *store, ev *event.Event) (reply []cmdReply) {
	var err error

	str := strings.TrimSpace(ev.Content.AsMessage().Body)
	str = strings.ToLower(str)

	args := strings.Split(str, " ")
	if len(args) < 0 {
		return
	}

	ud, err := data.user(ev.Sender)
	if err != nil {
		fmt.Println(err)
		reply = append(reply, cmdReply{
			"Oops, something went wrong", ""})
		return
	}

	var msg string
	var msgF string
	switch args[0] {
	case "events", "week":
		msg, err = cmdListEvents(ud)
	case "cal", "calendar":
		msg, err = cmdAddCalendar(ud, args)
	case "help":
		msg, msgF = cmdHelp()
	default:
		msg = "Unknown command"
	}

	if msg != "" {
		reply = append(reply, cmdReply{msg, msgF})
	}

	if err != nil {
		reply = append(reply, cmdReply{
			"Oops, something went wrong", ""})
		fmt.Println(err)
	}

	return
}

func cmdListEvents(u *user) (string, error) {
	cal, err := u.calendars[0].calendar()
	if err != nil {
		return "", err
	}

	now := time.Now()
	startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-int(now.Weekday())+1, 0, 0, 0, 0, now.Location())
	endOfWeek := startOfWeek
	endOfWeek = endOfWeek.Add(7 * 24 * time.Hour)

	events, err := cal.events(startOfWeek, endOfWeek)
	if err != nil {
		return "", err
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

	return msg, nil
}

func cmdAddCalendar(u *user, args []string) (string, error) {
	if len(args) < 2 {
		return "Provide the URI", nil
		// TODO: Improve message
	}

	uri := args[1]

	_, err := newCalDavCalendar(uri)
	if err != nil {
		fmt.Println(err)
		return "Specified URI is not a supported CalDAV calendar", nil
	}

	return "Calendar added", u.addCalendar(uri)
}

type helpSection struct {
	title string

	cmds []helpCommand
}

type helpCommand struct {
	cmd  string
	info string
}

var help = []helpSection{
	{
		"Managing your calendars",
		[]helpCommand{
			{"cal", "List your calendars"},
			{"cal add {name} {uri}", "Add a caldav calendar by name and URI"},
			{"cal remove {name}", "Remove the caldav calendar by name"},
		},
	},
	{
		"Viewing events in your calendars",
		[]helpCommand{
			{"week", "View your schedule for this week"},
		},
	},
}

func cmdHelp() (string, string) {
	lines := []string{}
	linesF := []string{}

	for i, s := range help {
		if i > 0 {
			lines = append(lines, "")
			linesF = append(linesF, "")
		}
		lines = append(lines, s.title)
		linesF = append(linesF, "<b>"+s.title+"</b><br />")

		linesF = append(linesF, "<ul>")

		for _, c := range s.cmds {
			msg := fmt.Sprintf("* %s  %s", c.cmd, c.info)
			msgF := fmt.Sprintf("<li>%s&nbsp;&nbsp;-&nbsp;&nbsp;%s</li>", c.cmd, c.info)

			lines = append(lines, msg)
			linesF = append(linesF, msgF)
		}

		linesF = append(linesF, "</ul>")
	}

	return strings.Join(lines, "\n"), strings.Join(linesF, "\n")
}
