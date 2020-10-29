package main

import (
	"fmt"
	"strconv"
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

	if !ud.ExistsInDB() {
		fmt.Println("Storing room")
		err = ud.store(ev.RoomID)
		if err != nil {
			fmt.Println(err)
			replies = append(replies, cmdReply{
				"Oops, something went wrong", ""})
		}
	}
	if ev.RoomID != ud.roomID {
		replies = append(replies, cmdReply{
			"This is not the room we normally use. Please go to: " + string(ud.roomID), ""})
	}

	var reply cmdReply
	switch args[0] {
	case "events", "week":
		yearNum := 0
		weekNum := 0

		var yearErr error
		var weekErr error

		if len(args) >= 2 {
			if len(args) >= 3 {
				yearNum, yearErr = strconv.Atoi(args[1])
				weekNum, weekErr = strconv.Atoi(args[2])
			} else {
				weekNum, weekErr = strconv.Atoi(args[1])
			}
		}

		if yearErr != nil || weekErr != nil {
			if yearErr != nil {
				replies = append(replies, cmdReply{"Invalid year specified", ""})
			}
			if weekErr != nil {
				replies = append(replies, cmdReply{"Invalid week specified", ""})
			}
			break
		}

		reply, err = cmdListEvents(ud, "week", yearNum, weekNum)
	case "today":
		reply, err = cmdListEvents(ud, "today", 0, 0)
	case "next":
		if len(args) < 2 {
			reply = cmdReply{"Next what? 'next week'?", ""}
			break
		}
		switch args[1] {
		case "week":
			reply, err = cmdListEvents(ud, "nextweek", 0, 0)
		}
	case "last", "prev", "previous":
		if len(args) < 2 {
			reply = cmdReply{"Last what? 'last week'?", ""}
			break
		}

		switch args[1] {
		case "week":
			reply, err = cmdListEvents(ud, "lastweek", 0, 0)
		}
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

func cmdListEvents(u *user, period string, year int, week int) (cmdReply, error) {
	cal, err := u.combinedCalendar()
	if err != nil {
		return cmdReply{}, err
	}

	now := time.Now()
	from := time.Time{}
	to := time.Time{}

	loc := now.Location() // TODO: loc should be depending on user.

	daysFromToTo := 7
	switch period {
	case "today":
		from = timeStartOfToday(now, loc)
		fmt.Println(from)
		daysFromToTo = 1
	case "nextweek":
		from = timeStartOfWeek(now, loc).AddDate(0, 0, 7)
	case "lastweek":
		from = timeStartOfWeek(now, loc).AddDate(0, 0, -7)
	default:
		if week == 0 {
			from = timeStartOfWeek(now, loc)
		} else {
			if year == 0 {
				year = now.Year()
			}
			from = timeStartOfYearPlusWeeks(year, loc, week)
		}
	}

	to = time.Date(from.Year(), from.Month(), from.Day()+daysFromToTo, 0, 0, 0, 0, loc)

	fmt.Println(from, to)

	events, err := cal.events(from, to)
	if err != nil {
		if err == errNoCalendars {
			return cmdReply{"You haven't configured any calendars. Use the 'cal add' command to start.", ""}, nil
		}
		return cmdReply{}, err
	}

	// TODO: Properly handle multi-day events.

	lines := []string{}
	linesF := []string{}

	if strings.Contains(period, "week") {
		_, week := from.ISOWeek()
		wk := strconv.Itoa(week)

		lines = append(lines, "Week "+wk, "")
		linesF = append(linesF, "<b>Week "+wk+"</b>", "")
	}

	days := events.format()
	for i, day := range days {
		if to.Before(day.day) {
			continue
		}

		if i > 0 {
			lines = append(lines, "")
			linesF = append(linesF, "")
		}

		header := day.day.Format("Monday 2 January")
		lines = append(lines, fmt.Sprintf("%s", header))
		linesF = append(linesF, fmt.Sprintf("<b>%s</b>", header))

		for _, ev := range day.events {
			lines = append(lines, fmt.Sprintf("%s - %s: %s", ev.from.Format("15:04"), ev.to.Format("15:04"), ev.text))
			linesF = append(linesF, fmt.Sprintf("<code>%s - %s</code>: %s", ev.from.Format("15:04"), ev.to.Format("15:04"), ev.text))
		}
	}

	return cmdReply{strings.Join(lines, "\n"), strings.Join(linesF, "<br />")}, nil
}

func timeStartOfToday(base time.Time, loc *time.Location) time.Time {
	return time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, loc)
}

func timeStartOfWeek(base time.Time, loc *time.Location) time.Time {
	return time.Date(base.Year(), base.Month(), base.Day()-int(base.Weekday())+1, 0, 0, 0, 0, loc)
}

func timeStartOfYearPlusWeeks(year int, loc *time.Location, weekNumber int) time.Time {
	tm := time.Date(year, 1, 1, 0, 0, 0, 0, loc)

	addDays := (weekNumber - 1) * 7
	addDays += int(tm.Weekday()-time.Monday) * -1

	tm = tm.AddDate(0, 0, addDays)

	return tm
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
		{"week {number}", "View your schedule for the specified week", ""},
		{"week {year} {number}", "View your schedule for the specified week", ""},
		{"last week", "View your schedule for last week", ""},
		{"next week", "View your schedule for next week", ""},
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
		msg := fmt.Sprintf("* %s - %s", c.cmd, c.info)
		msgF := fmt.Sprintf("&nbsp;&#9702; <code>%s</code>&nbsp;&nbsp;-&nbsp;&nbsp;%s", c.cmd, c.info)

		lines = append(lines, msg)
		linesF = append(linesF, msgF)
	}

	return cmdReply{strings.Join(lines, "\n"), strings.Join(linesF, "<br />\n")}
}

var usageCalAdd = helpCommand{
	"cal add {name} {type} {address}",
	"Add a calendar by choosing a name, and specifying the type (caldav or ical) and webaddress",
	"cal add personal caldav https://mysite.nl/calendar/3owevfu1d0rb3psw",
}

var usageCalRemove = helpCommand{
	"cal remove {name}",
	"Remove the specified calendar from the bot",
	"",
}

func formatUsage(usage helpCommand) cmdReply {
	msg := fmt.Sprintf("Usage: %s\n%s", usage.cmd, usage.info)
	msgF := fmt.Sprintf("<b>Usage</b>: %s<br />\n%s", usage.cmd, usage.info)
	if usage.example != "" {
		msg += "\n\nExample: " + usage.example
		msgF += "<br />\n<br />\n<b>Example</b>: " + usage.example
	}
	return cmdReply{msg, msgF}
}
