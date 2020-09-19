package main

import (
	"flag"
	"fmt"
	"os"

	"maunium.net/go/mautrix/id"
)

func main() {
	fmt.Println("1")

	db, err := initSQLDB("matrix-caldav-bot.db")
	if err != nil {
		panic(err)
	}

	dbid, err := db.addCalendar(id.UserID("@remi:remi.im"), "https://ijsbeer.nl")
	fmt.Println(dbid, err)

	cals, err := db.fetchCalendars(id.UserID("@remi:remi.im"))
	fmt.Println(cals, err)

	cfgFileName := flag.String("config-filename", "config.json", "")
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

	initMatrixBot(cfg.MatrixBot, newDataStore())

	fmt.Println("Started")

	<-make(chan struct{})
}
