package main

import (
	"encoding/json"
	"log"
	"os"
	"path"

	"github.com/juubisnake/concourse-job-log-resource/pkg/fly"
	"github.com/juubisnake/concourse-job-log-resource/pkg/lib"
)

func main() {
	var msg lib.Message
	if err := json.NewDecoder(os.Stdin).Decode(&msg); err != nil {
		log.Fatalf("error while unmarshaling input to json: %s", err.Error())
	}

	flyClient := fly.New(msg.Check)
	if err := flyClient.Login(); err != nil {
		log.Fatalf("error while logging into fly: %s", err.Error())
	}
	logs, err := flyClient.ScrapeLogs(msg.Version.APIURL)
	if err != nil {
		log.Fatalf("error while scraping logs: %s", err.Error())
	}

	dir := os.Args[1]
	f, err := os.Create(path.Join(dir, "logs.txt"))
	if err != nil {
		log.Fatalf("error while creating log file: %s", err.Error())
	}
	defer f.Close()

	if _, err := f.WriteString(logs); err != nil {
		log.Fatalf("error while writing to log file: %s", err.Error())
	}

	out := &struct {
		Version  *lib.Version   `json:"version"`
		Metadata []lib.Metadata `json:"metadata"`
	}{
		Version:  msg.Version,
		Metadata: []lib.Metadata{},
	}

	if err := json.NewEncoder(os.Stdout).Encode(&out); err != nil {
		log.Fatalf("error while marshalling build metadata into json")
	}
}
