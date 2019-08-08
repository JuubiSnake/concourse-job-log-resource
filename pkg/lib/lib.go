package lib

import "github.com/juubisnake/concourse-job-log-resource/pkg/fly"

type Message struct {
	Check   *fly.Creds `json:"source"`
	Version *Version   `json:"version"`
}

type Version struct {
	ID     string `json:"id"`
	APIURL string `json:"api_uri"`
}

type Metadata struct{}
