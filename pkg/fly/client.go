package fly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/pborman/ansi"
)

const (
	BUILD_LATEST_VERSION = "latest"

	fly = "fly --target concourse"
)

type Creds struct {
	URI         string `json:"uri"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Team        string `json:"team"`
	Pipeline    string `json:"pipeline"`
	Job         string `json:"job"`
	HasFinished bool   `json:"finished"`
}

type buildDataList []buildData

func (s buildDataList) Len() int {
	return len(s)
}
func (s buildDataList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s buildDataList) Less(i, j int) bool {
	return s[i].Data.Time < s[j].Data.Time
}

type Build struct {
	ID     int    `json:"id"`
	APIURL string `json:"api_url"`
}

type buildData struct {
	Data buildPayload `json:"data"`
}

type buildPayload struct {
	Payload string      `json:"payload"`
	Origin  buildOrigin `json:"origin"`
	Time    int         `json:"time"`
}

type buildOrigin struct {
	ID string `json:"id"`
}

type Client struct {
	creds *Creds
}

func New(creds *Creds) *Client {
	c := &Client{
		creds: creds,
	}
	c.creds.Pipeline = strings.Replace(c.creds.Pipeline, " ", "\\ ", -1)
	c.creds.Job = strings.Replace(c.creds.Job, " ", "\\ ", -1)
	return c
}

func (c *Client) Login() error {
	cmd := fmt.Sprintf(
		"%s login --team-name %s --concourse-url %s -u %s -p %s",
		fly,
		c.creds.Team,
		c.creds.URI,
		c.creds.Username,
		c.creds.Password,
	)
	if _, err := execute(cmd).Run(); err != nil {
		return err
	}
	return nil
}

type buildsList struct {
	builds []Build
	target int
}

func (c *Client) FindBuild() (*Build, error) {
	cmd := fmt.Sprintf("%s curl /api/v1/teams/%s/pipelines/%s/jobs/%s/builds", fly, c.creds.Team, c.creds.Pipeline, c.creds.Job)
	out, err := execute(cmd).Run()
	if err != nil {
		return nil, err
	}
	var builds []Build
	if err := json.Unmarshal(out, &builds); err != nil {
		return nil, err
	}
	return c.findLatest(builds), nil
}

func (c *Client) ScrapeLogs(apiURL string) (string, error) {
	cmd := fmt.Sprintf("%s curl %s/events --print-and-exit", fly, apiURL)
	out, err := execute(cmd).Run()
	if err != nil {
		return "", err
	}
	r, err := regexp.Compile(`"Authorization: (Bearer [\w|\W]+)"`)
	if err != nil {
		return "", err
	}

	// first match will be entire regexp - we want the bearer token only
	token := r.FindStringSubmatch(string(out))[1]

	eventsURL := fmt.Sprintf("%s/%s/events", c.creds.URI, apiURL)
	req, err := http.NewRequest("GET", eventsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", token)

	stream, err := eventsource.SubscribeWithRequest("", req)
	if err != nil {
		return "", err
	}

	sb := new(strings.Builder)
	logmap := make(map[string]buildDataList)
	for {
		select {
		case e := <-stream.Events:
			var bd buildData
			if e.Data() == "" {
				continue
			}
			if err := json.Unmarshal([]byte(e.Data()), &bd); err != nil {
				return "", err
			}
			if bd.Data.Payload == "" {
				continue
			}
			if logmap[bd.Data.Origin.ID] == nil {
				logmap[bd.Data.Origin.ID] = []buildData{}
			}
			logmap[bd.Data.Origin.ID] = append(logmap[bd.Data.Origin.ID], bd)

		case <-time.After(time.Second * 5):
			for k, v := range logmap {
				sort.Sort(v)
				if _, err := sb.WriteString(fmt.Sprintf("\n==============================| %s - START |==============================\n", k)); err != nil {
					return "", err
				}
				for i := range v {
					fmtdPayload, err := ansi.Strip([]byte(v[i].Data.Payload))
					if err != nil {
						return "", err
					}
					t := time.Unix(int64(v[i].Data.Time), 0)
					logEntry := fmt.Sprintf("%s:\n%s", t.String(), string(fmtdPayload))
					if _, err := sb.WriteString(logEntry); err != nil {
						return "", err
					}
				}
				if _, err := sb.WriteString(fmt.Sprintf("\n==============================|  %s - END  |==============================\n", k)); err != nil {
					return "", err
				}
			}
			return sb.String(), nil
		}
	}
}

func (c *Client) findLatest(builds []Build) *Build {
	var latest *Build
	for i := range builds {
		if latest == nil {
			latest = &builds[i]
		} else if latest.ID < builds[i].ID {
			latest = &builds[i]
		}
	}
	return latest
}

type executor struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	cmd    *exec.Cmd
}

func execute(command string) *executor {
	cmd := exec.Command("sh", "-exc", command)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return &executor{
		stdout: stdout,
		stderr: stderr,
		cmd:    cmd,
	}
}

func (e *executor) Run() ([]byte, error) {
	if err := e.cmd.Run(); err != nil {
		fmtdErr := fmt.Errorf("%s - %s", e.stderr.String(), err.Error())
		return nil, fmtdErr
	}
	return e.stdout.Bytes(), nil
}
