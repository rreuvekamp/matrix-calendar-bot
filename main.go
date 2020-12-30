package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	cfgFileName := flag.String("config", "config.json", "")
	flag.Parse()

	cfg, isNew, err := loadConfig(*cfgFileName)
	if err != nil {
		fmt.Printf("Could not read JSON from configuration file with name %s. "+
			"Could not create it either.\nError message: %s.\n",
			*cfgFileName, err.Error())
		os.Exit(1)
		return
	}
	if isNew {
		fmt.Printf("Configuration file created. Saved as %s. Please look at it, fill in the values and rerun the application.\n", *cfgFileName)
		os.Exit(2)
		return
	}

	db, err := initSQLDB(cfg.SQLiteURI)
	if err != nil {
		fmt.Println("Error initialising database:", err)
		os.Exit(4)
	}

	data := newDataStore(db)

	fmt.Println("Reading database into memory...")

	err = data.populateFromDB()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Initialising Matrix bot...")

	m, err := initMatrixBot(cfg.MatrixBot, data)
	if err != nil {
		fmt.Println("Error initialising Matrix connection:", err)
		os.Exit(3)
	}

	fmt.Println("Setting up reminder timers...")

	if true {
		setupReminderTimers(m, data)
	}

	fmt.Println("Done")

	<-make(chan struct{})
}

func setupReminderTimers(m matrixBot, data *store) {
	for _, user := range data.users {
		send := func(ev *calendarEvent) {
			msg := ""

			timeUntil := ev.from.Sub(time.Now())

			if timeUntil.Minutes() > 0 {
				msg = fmt.Sprintf("Reminder: %q starts in %d minutes", ev.text, int(timeUntil.Minutes()))
			} else {
				msg = fmt.Sprintf("Reminder: %q starts now", ev.text)
			}
			m.sendMessage(user.roomID, msg, "")
		}

		go func() {
			err := user.initialiseReminderTimer(send, 65*time.Minute)
			if err != nil {
				fmt.Println(err)
			}

			for {
				<-time.After(60 * time.Minute)
				fmt.Println("call setup reminder timers")
				err = user.restartReminderTimer()
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("done call setup reminder timers")
			}
		}()
		<-time.After(100 * time.Millisecond)
	}
}
