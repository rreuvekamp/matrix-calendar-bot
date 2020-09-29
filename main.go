package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
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

	cli, err := initMatrixBot(cfg.MatrixBot, data)
	if err != nil {
		fmt.Println("Error initialising Matrix connection:", err)
		os.Exit(3)
	}

	fmt.Println("Setting up reminder timers...")

	setupReminderTimers(cli, data)

	fmt.Println("Done")

	<-make(chan struct{})
}

func setupReminderTimers(cli *mautrix.Client, data *store) {
	send := func(ev calendarEvent) {
		sendMessage(cli, id.RoomID("!qvPycavGoabBgSxiDz:remi.im"), "Reminder for: "+ev.text, "")
	}

	for _, user := range data.users {
		go func() {
			for {
				user.setupReminderTimer(send)
				<-time.After(30 * time.Minute)
			}
		}()
		<-time.After(100 * time.Millisecond)
	}
}
