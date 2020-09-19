package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
)

type config struct {
	MatrixBot configMatrixBot
}

type configMatrixBot struct {
	Homeserver string
	AccountID  string
	Token      string
}

type loadConfigError struct {
	create bool  // If the error occured while trying to create the file.
	err    error // Original error.
}

func (l loadConfigError) Error() string {
	return "cannot load config; " + l.err.Error()
}

var defaultConfig = config{
	MatrixBot: configMatrixBot{
		Homeserver: "https://example.org",
		AccountID:  "@calendarbot:remi.im",
		Token:      "",
	},
}

// loadConfig unmarhsals the contents of the file with given filename as JSON, which
// is returned.
// If there is no file with the given filename the file will be created containing
// defaultConfig as JSON. isNew is then set to true.
func loadConfig(filename string) (cfg config, isNew bool, err error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		if !os.IsNotExist(err) {
			return config{}, false, loadConfigError{false, err}
		}

		// Configuration file probably doesn't exist.
		// Create it with defaultConfig in JSON as contents.
		err = createConfig(filename, defaultConfig)
		if err != nil {
			return defaultConfig, true, loadConfigError{true, err}
		}

		return defaultConfig, true, nil
	}

	dec := json.NewDecoder(f)
	err = dec.Decode(&cfg)
	if err != nil && err != io.EOF {
		return cfg, isNew, loadConfigError{false, err}
	}

	return cfg, isNew, err
}

func createConfig(filename string, cfg config) (err error) {
	data, err := json.MarshalIndent(cfg, "", "	")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, data, 0600)
	if err != nil {
		return err
	}

	return nil
}
