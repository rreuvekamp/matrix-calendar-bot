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

func handleCommand(cli *mautrix.Client, data *store, ev *event.Event) (replies []cmdReply) {
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
		replies = append(replies, cmdReply{
			"Oops, something went wrong", ""})
		return
	}

	var reply cmdReply
	switch args[0] {
	case "events", "week":
		reply.msg, err = cmdListEvents(ud)
	case "cal", "calendar":
		if len(args) < 2 {
			reply = formatHelp(helpCal)
			break
		}

		switch args[1] {
		case "add":
			reply, err = cmdCalendarAdd(ud, args)
		default:
			reply = formatHelp(helpCal)
		}
	case "help":
		reply = formatAllHelp()
	default:
		reply.msg = "Unknown command"
	}

	if reply.msg != "" {
		replies = append(replies, reply)
	}

	if err != nil {
		replies = append(replies, cmdReply{
			"Oops, something went wrong", ""})
		fmt.Println(err)
	}

	return
}

func cmdListEvents(u *user) (string, error) {
	cal, err := u.combinedCalendar()
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

func cmdCalendarAdd(u *user, args []string) (cmdReply, error) {
	if len(args) < 4 {
		return formatUsage(usageCalAdd), nil
		// TODO: Improve message
	}

	//name := args[2]

	uri := args[3]

	_, err := newICalCalendar(uri)
	if err != nil {
		fmt.Println(err)
		return cmdReply{"Specified URI is not a supported CalDAV calendar", ""}, nil
	}

	return cmdReply{"Calendar added", ""}, u.addCalendar(uri)
}

type helpSection struct {
	title string

	cmds []helpCommand
}

type helpCommand struct {
	cmd     string
	info    string
	example string
}

var helpCal = helpSection{
	"Managing your calendars",
	[]helpCommand{
		{"cal", "List your calendars", ""},
		{"cal add {name} {uri}", "Add a CalDav calendar by name and URI", ""},
		{"cal remove {name}", "Remove the CalDav calendar by name", ""},
	},
}
var helpView = helpSection{
	"Viewing events in your calendars",
	[]helpCommand{
		{"week", "View your schedule for this week", ""},
	},
}

func formatAllHelp() cmdReply {
	lines := []string{}
	linesF := []string{}

	for _, s := range []helpSection{helpCal, helpView} {
		reply := formatHelp(s)
		lines = append(lines, reply.msg)
		linesF = append(linesF, reply.msgF)
	}

	return cmdReply{strings.Join(lines, "\n"), strings.Join(linesF, "\n")}
}

func formatHelp(help helpSection) cmdReply {
	lines := []string{}
	linesF := []string{}

	lines = append(lines, help.title)
	linesF = append(linesF, "<b>"+help.title+"</b><br />")

	linesF = append(linesF, "<ul>")

	for _, c := range help.cmds {
		msg := fmt.Sprintf("* %s  %s", c.cmd, c.info)
		msgF := fmt.Sprintf("<li>%s&nbsp;&nbsp;-&nbsp;&nbsp;%s</li>", c.cmd, c.info)

		lines = append(lines, msg)
		linesF = append(linesF, msgF)
	}

	linesF = append(linesF, "</ul>")

	return cmdReply{strings.Join(lines, "\n"), strings.Join(linesF, "\n")}
}

var usageCalAdd = helpCommand{
	"cal add {name} {uri}",
	"Add a CalDav calendar by specifying the CalDav URI and a calendar name of your choice",
	"cal add ",
}

func formatUsage(usage helpCommand) cmdReply {
	msg := fmt.Sprintf("Usage: %s\n%s", usage.cmd, usage.info)
	msgF := fmt.Sprintf("<b>Usage</b>: %s<br />\n%s", usage.cmd, usage.info)
	return cmdReply{msg, msgF}
}
