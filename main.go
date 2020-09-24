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

	cli, err := initMatrixBot(cfg.MatrixBot, newDataStore(db))
	if err != nil {
		fmt.Println("Error initialising Matrix connection:", err)
		os.Exit(3)
	}

	go setupReminders(cli, db)

	fmt.Println("Started")

	<-make(chan struct{})
}

func setupReminders(cli *mautrix.Client, db *sqlDB) {
	cals, err := db.fetchAllCalendars()
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, uc := range cals {
		cal, err := uc.calendar()
		if err != nil {
			fmt.Println(err)
			continue
		}

		evs, err := cal.events(time.Now(), time.Now().Add(24*time.Hour))
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, event := range evs {
			ev := event
			go func() {
				fmt.Println("Setup reminder for:", ev.text)
				<-time.After(time.Until(ev.from))
				sendMessage(cli, id.RoomID("!qvPycavGoabBgSxiDz:remi.im"), "Reminder for:"+ev.text)
			}()
		}

		<-time.After(100 * time.Millisecond)
	}
}
