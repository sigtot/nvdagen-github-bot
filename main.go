package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var secret []byte

const repo = "Nettverkedagen-2"
const masterBranch = "master"
const requestLogFileName = "requests.log"
const webhookLogFileName = "webhooks.log"
const port = 8000

func main() {
	log.SetFlags(0)
	requestLogFile, err := os.OpenFile(requestLogFileName, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0644)
	okOrPanic(err)
	defer func() {
		err := requestLogFile.Close()
		okOrPanic(err)
	}()

	webhookLogFile, err := os.OpenFile(webhookLogFileName, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0644)
	okOrPanic(err)
	defer func() {
		err := requestLogFile.Close()
		okOrPanic(err)
	}()

	requestLogger := log.New(requestLogFile, "", log.LstdFlags)
	webhookLogger := log.New(webhookLogFile, "", 0)

	pushEvents := make(chan *github.PushEvent)
	secret = []byte(os.Getenv("GITHUB_WEBHOOKS_SECRET"))
	http.HandleFunc("/github", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, requestLogger, pushEvents)
	})
	go handlePushEvents(webhookLogger, pushEvents)

	fmt.Printf("Server listening on port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		panic(err)
	}
	fmt.Println("Should not get here")
}

func okOrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request, logger *log.Logger, pushEvents chan<- *github.PushEvent) {
	fmt.Printf("new request\n")
	payload, err := github.ValidatePayload(r, secret)
	if err != nil {
		logger.Printf("Payload validation failed: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		logger.Printf("Failed to parse payload %s\n", err)
		return
	}

	pushEvent, ok := event.(*github.PushEvent)
	if !ok {
		log.Println("Not a push event")
		return
	}

	if *pushEvent.Repo.Name != repo {
		log.Printf("Bad webhook: Expected repo %s but got %s\n", repo, *pushEvent.Repo.Name)
		w.WriteHeader(http.StatusBadRequest) // Other repos shouldn't show up here, so something is funky
		return
	}

	pushEvents <- pushEvent
}

func handlePushEvents(logger *log.Logger, pushEvents <-chan *github.PushEvent) {
	for {
		pushEvent := <-pushEvents
		func() {
			logger.Println("\n-------------------- RECEIVED WEBHOOK --------------------")
			logger.Printf("Time: %s\n", time.Now().String())
			defer logger.Println("----------------------------------------------------------")

			if branch := strings.Split(*pushEvent.Ref, "/")[2]; branch != masterBranch {
				logger.Printf("Branch is %s and not %s, omitting deploy\n", masterBranch, branch)
				return
			}

			logger.Printf("New commit on %s detected. Starting deploy...\n", masterBranch)
		}()
	}
}
