package main

import (
	"encoding/json"
	"os"
)

type Configuration struct {
	DB DBSettings `json:"database"`
}

type DBSettings struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

// getConfigFromFile does what it says on the box and returns a Configuration object
// representing the config file.
func getConfigFromFile(path string) (*Configuration, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err = decoder.Decode(&configuration)
	if err != nil {
		return nil, err
	}

	return &configuration, nil
}
