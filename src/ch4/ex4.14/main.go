package main

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

var (
	log, _    = zap.NewDevelopment()
	githubApi = "https://api.github.com"
)

type User struct {
	Login   string
	HtmlUrl string `json:"html_url"`
}

type Issue struct {
	Number    int
	Id        int
	HtmlUrl   string `json:"html_url"`
	Title     string
	State     string
	CreatedAt time.Time `json:"created_at"`
	User      *User
	Assignee  *User
	Body      string
}

type Milestones struct {
	Number      int
	Id          int
	HtmlUrl     string `json:"html_url"`
	Title       string
	Description string
	Creator     *User
	State       string
	CreatedAt   time.Time `json:"created_at"`
}

type Kind string
type GhApiRepoCache struct {
	Repo string
}

func (c *GhApiRepoCache) Issues() ([]Issue, error) {
	data, err := c.get("issues")
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal([]byte(data), &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

func (c *GhApiRepoCache) Milestones() ([]Milestones, error) {
	data, err := c.get("milestones")
	if err != nil {
		return nil, err
	}

	var milestones []Milestones
	if err := json.Unmarshal([]byte(data), &milestones); err != nil {
		return nil, err
	}
	return milestones, nil
}

func (c *GhApiRepoCache) get(kind Kind) (string, error) {
	cacheFile := fmt.Sprintf("./data/%s/%s.json", c.Repo, kind)

	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		// When the file doesn't exist, fetch its data and save it to the file.
		// Create the parent directories of the file
		if err := os.MkdirAll(filepath.Dir(cacheFile), 0775); err != nil {
			log.Error("Failed to create parent directories", zap.String("dir", cacheFile), zap.Error(err))
			return "", err
		}

		file, err := os.Create(cacheFile)
		if err != nil {
			return "", err
		}

		stream, err := doHttpGetStreaming(c.getUri(kind))
		if err != nil {
			return "", err
		}

		// TODO(ljdelight): lookup how to defer a func call, it would be nice to log if it fails
		defer stream.Close()

		if _, err := io.Copy(file, stream); err != nil {
			// TODO(ljdelight): cached file very likely to be incomplete content, so prooooobably delete it
			return "", err
		}
	} else if err != nil {
		// The stat failed in some way (other than the file doesn't exist)
		return "", err
	}

	// The cached data exists. Return its contents.
	res, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func (c *GhApiRepoCache) getUri(kind Kind) string {
	return fmt.Sprintf("%s/repos/%s/%s", githubApi, c.Repo, kind)
}

// TODO(ljdelight): using the easy route atm. lots of uris have paging, and this simple approach does not support it.
// Perform an http get on the URI and return the `response.Body`. Callers are expected to `Close` the io.ReadCloser.
// The intent is to stream the data back without loading it all in memory.
func doHttpGetStreaming(uri string) (io.ReadCloser, error) {
	log.Info("GET request", zap.String("uri", uri))
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	// TODO(ljdelight): Need to process the http return values (a 404 could have a body...)
	return resp.Body, nil
}

func main() {
	log.Info("Starting execution")
	gh := GhApiRepoCache{Repo: "kubernetes-sigs/kubebuilder"}

	issues, err := gh.Issues()
	if err != nil {
		log.Warn("Failed to get issues", zap.Error(err))
	}
	log.Info("Found issues", zap.Any("issues", issues))

	const templIssue = `------------------
Number:   {{.Number}}
Title:    {{.Title}}
State:    {{.State}}
User:     {{.User.Login}}
Assignee: {{if .Assignee}}{{.Assignee.Login}}{{else}}Unassigned{{end}}
`
	report, err := template.New("issueReport").Parse(templIssue)
	if err != nil {
		log.Error("Failed to generate issues report", zap.Error(err))
	}
	for _, issue := range issues[0:10] {
		if err = report.Execute(os.Stdout, issue); err != nil {
			log.Error("Failed to execute report", zap.Error(err))
		}
	}

	milestones, err := gh.Milestones()
	if err != nil {
		log.Warn("Failed to get milestones", zap.Error(err))
	}
	log.Info("Found milestones", zap.Any("milestones", milestones))

	const templMilestone = `------------------
Number:      {{.Number}}
Title:       {{.Title}}
Description: {{.Description}}
State:       {{.State}}
Creator:     {{.Creator.Login}}
Created:     {{.CreatedAt}}	
`
	reportMilestone, err := template.New("milestoneReport").Parse(templMilestone)
	if err != nil {
		log.Error("Failed to generate issues report", zap.Error(err))
	}
	for _, milestone := range milestones {
		if err = reportMilestone.Execute(os.Stdout, milestone); err != nil {
			log.Error("Failed to execute report", zap.Error(err))
		}
	}
}
