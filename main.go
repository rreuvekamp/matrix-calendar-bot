package main

import (
	"flag"
	"fmt"
	"os"
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

	ds := newDataStore(db)

	fmt.Println("Reading database into memory...")

	err = ds.populateFromDB()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Initialising Matrix bot...")

	cli, err := initMatrixBot(cfg.MatrixBot, ds)
	if err != nil {
		fmt.Println("Error initialising Matrix connection:", err)
		os.Exit(3)
	}

	fmt.Println("Setting up reminder timers...")

	for _, user := range ds.users {
		user.setupReminderTimer(cli)
	}

	fmt.Println("Done")

	<-make(chan struct{})
}
