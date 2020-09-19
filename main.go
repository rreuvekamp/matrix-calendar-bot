package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
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

	_ = cfg

	initMatrixBot(cfg.MatrixBot)

	fmt.Println("Started")

	<-make(chan struct{})
}
