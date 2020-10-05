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
			reply = cmdCalendarList(ud)
			break
		}

		switch args[1] {
		case "add":
			reply, err = cmdCalendarAdd(ud, args)
		case "list":
			reply = cmdCalendarList(ud)
		case "remove":
			reply, err = cmdCalendarRemove(ud, args)
		default:
			replies = append(replies, cmdReply{
				"Unknown option", ""})
			reply = formatHelp(helpCal)
		}
	case "help", "?":
		reply = formatAllHelp()
	default:
		replies = append(replies, cmdReply{"Unknown command", ""})
		reply = formatAllHelp()
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
		if err == errNoCalendars {
			return "You haven't configured any calendars. Use the 'cal add' command to start.", nil
		}
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

func cmdCalendarRemove(u *user, args []string) (cmdReply, error) {
	if len(args) < 3 {
		return formatUsage(usageCalRemove), nil
	}

	name := strings.ToLower(args[2])

	err := u.removeCalendar(name)
	if err != nil {
		if err == errCalendarNotExists {
			return cmdReply{
				"There is no calendar named " + name,
				"There is no calendar named <b>" + name + "</b>"}, nil
		}
		return cmdReply{}, err
	}
	return cmdReply{"Calendar " + name + " removed",
		"Calendar <b>" + name + "</b> removed"}, nil
}

var replyNoCalendars = cmdReply{"You haven't configured any calendars. Use the 'cal add' command to start.", ""}

func cmdCalendarList(u *user) cmdReply {
	lines := []string{}
	linesF := []string{}

	if len(u.calendars) == 0 {
		return replyNoCalendars
	}

	u.calendarsMutex.RLock()
	amount := len(u.calendars)
	u.calendarsMutex.RUnlock()

	if amount == 1 {
		lines = append(lines, "You have one calendar")
		linesF = append(linesF, "You have <b>one</b> calendar")
	} else {
		lines = append(lines, fmt.Sprintf("You have %d calendars", amount))
		linesF = append(linesF, fmt.Sprintf("You have <b>%d</b> calendars", amount))
	}

	lines = append(lines, "")
	linesF = append(linesF, "")

	u.calendarsMutex.RLock()
	for i, uc := range u.calendars {
		if i > 0 {
			lines = append(lines, "")
			linesF = append(linesF, "")
		}

		lines = append(lines, uc.Name)
		lines = append(lines, "type: "+string(uc.CalType))

		linesF = append(linesF, fmt.Sprintf("<b>%s</b>", uc.Name))
		linesF = append(linesF, "type: "+string(uc.CalType))
	}
	u.calendarsMutex.RUnlock()

	return cmdReply{strings.Join(lines, "\n"), strings.Join(linesF, "<br />")}
}

func cmdCalendarAdd(u *user, args []string) (cmdReply, error) {
	if len(args) < 5 {
		return formatUsage(usageCalAdd), nil
	}

	name := strings.ToLower(args[2])

	if u.hasCalendar(name) {
		return cmdReply{
			"You already have a calendar named " + name + ". Please choose a different name",
			"You already have a calendar named <b>" + name + "</b>. Please choose a different name."}, nil
	}

	calTypeStr := strings.ToLower(args[3])

	uri := args[4]

	var calType calendarType
	if calTypeStr == "caldav" {
		calType = calendarTypeCalDav

		_, err := newCalDavCalendar(uri)
		if err != nil {
			fmt.Println(err)
			return cmdReply{"Specified address is not a supported CalDAV calendar", ""}, nil
		}
	} else if calTypeStr == "ical" {
		calType = calendarTypeICal

		_, err := newICalCalendar(uri)
		if err != nil {
			fmt.Println(err)
			return cmdReply{"Specified address is not a supported ical calendar", ""}, nil
		}
	} else {
		return cmdReply{"Invalid calendar type specified. Supported types are 'caldav' and 'ical'.", ""}, nil
	}

	return cmdReply{"Calendar added", ""}, u.addCalendar(name, calType, uri)
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
		usageCalAdd,
		usageCalRemove,
	},
}
var helpView = helpSection{
	"Viewing events in your calendars",
	[]helpCommand{
		{"week", "View your schedule for this week", ""},
	},
}

func formatAllHelp() cmdReply {
	lines := []string{"Use these commands to interact with the bot", ""}
	linesF := []string{"<b>Use these commands to interact with the bot</b>", ""}

	for i, s := range []helpSection{helpCal, helpView} {
		if i > 0 {
			lines = append(lines, "")
			linesF = append(linesF, "")
		}
		reply := formatHelp(s)
		lines = append(lines, reply.msg)
		linesF = append(linesF, reply.msgF)
	}

	return cmdReply{strings.Join(lines, "\n"), strings.Join(linesF, "<br />\n")}
}

func formatHelp(help helpSection) cmdReply {
	lines := []string{}
	linesF := []string{}

	lines = append(lines, help.title)
	linesF = append(linesF, "<b>"+help.title+"</b>")

	for _, c := range help.cmds {
		msg := fmt.Sprintf("* %s  %s", c.cmd, c.info)
		msgF := fmt.Sprintf("&nbsp;&#9702; %s&nbsp;&nbsp;-&nbsp;&nbsp;%s", c.cmd, c.info)

		lines = append(lines, msg)
		linesF = append(linesF, msgF)
	}

	return cmdReply{strings.Join(lines, "\n"), strings.Join(linesF, "<br />\n")}
}

var usageCalAdd = helpCommand{
	"cal add {name} {type} {address}",
	"Add a CalDav calendar by chosing a name, specifying the type (caldav or ical) and specifying the address",
	"cal add personal caldav https://mysite.nl/calendar/3owevfu1d0rb3psw",
}

var usageCalRemove = helpCommand{
	"cal remove {name}",
	"Remove specified calendar from the bridge",
	"",
}

func formatUsage(usage helpCommand) cmdReply {
	msg := fmt.Sprintf("Usage: %s\n%s", usage.cmd, usage.info)
	msgF := fmt.Sprintf("<b>Usage</b>: %s<br />\n%s", usage.cmd, usage.info)
	if usage.example != "" {
		msg += "\nExample: " + usage.example
		msgF += "<br />\n<b>Example</b>: " + usage.example
	}
	return cmdReply{msg, msgF}
}
