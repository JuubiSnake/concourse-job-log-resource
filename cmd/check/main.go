package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/juubisnake/concourse-job-log-resource/pkg/fly"
	"github.com/juubisnake/concourse-job-log-resource/pkg/lib"
)

func main() {
	var msg lib.Message
	if err := json.NewDecoder(os.Stdin).Decode(&msg); err != nil {
		log.Fatalf("error while unmarshaling message from stdin: %s", err.Error())
	}

	if msg.Version == nil {
		msg.Version = &lib.Version{ID: fly.BUILD_LATEST_VERSION, APIURL: ""}
	}

	flyClient := fly.New(msg.Check)

	if err := flyClient.Login(); err != nil {
		log.Fatalf("error while communicating with targeted concourse: %s", err.Error())
	}

	build, err := flyClient.FindBuild()
	if err != nil {
		log.Fatalf("error while trying to list builds: %s", err.Error())
	}

	if build == nil {
		log.Fatalf("unable to find build from requested job: %s", msg.Check.Job)
		return
	}

	if build.APIURL == msg.Version.APIURL {
		fmt.Print("[]")
		return
	}

	out := []*lib.Version{
		&lib.Version{
			ID:     strconv.Itoa(build.ID),
			APIURL: build.APIURL,
		},
	}

	if err := json.NewEncoder(os.Stdout).Encode(&out); err != nil {
		log.Fatalf("error while marshalling build result into json")
	}
}
