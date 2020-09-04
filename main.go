package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	work_messages "github.com/moraisworkrunner/work-messages"
	"google.golang.org/protobuf/proto"
)

var (
	retryCountHeader = "X-CloudTasks-TaskExecutionCount"
)

func handler(w http.ResponseWriter, r *http.Request) {
	workResponse := work_messages.SvcWorkResponse{}
	target := os.Getenv("NOTIFIER_QUEUE")
	if target == "" {
		target = "queue"
	}
	location := os.Getenv("NOTIFIER_LOCATION")
	if location == "" {
		location = "nowhere"
	}
	maxAttemptsStr := os.Getenv("MAX_ATTEMPTS")
	if maxAttemptsStr == "" {
		maxAttemptsStr = "20"
	}
	maxAtt, err := strconv.Atoi(maxAttemptsStr)
	if err != nil {
		maxAtt = 20
	}
	maxAttempts := int64(maxAtt)
	// Where to send, and who will service, problematic requests
	problematicQueue := os.Getenv("PROBLEM_QUEUE")
	problematicService := os.Getenv("PROBLEM_SERVICE")

	// Parse the body and protobuf message from the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		workResponse.Error = &work_messages.Error{
			Message: fmt.Sprintf("Failed to read request"),
		}
		log.Printf("Failed to read work request: %v\n", r.Body)
		// Do not retry
		return
	}
	in := &work_messages.SvcWorkRequest{}
	if err := proto.Unmarshal(body, in); err != nil {
		workResponse.Error = &work_messages.Error{
			Message: fmt.Sprintf("Failed to parse work request: %s", err),
		}
		log.Printf("Failed to unmarshal proto: %v\n", body)
		// Do not retry
		return
	}
	workResponse.Context = in.Context
	webhookURL := in.WebhookUrl

	// Do something with the payload, and return the appropriate status
	if err := processWork(in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		workResponse.Error = &work_messages.Error{
			Message: err.Error(),
		}
		// "maxAttempts" may be leveraged to push problematic work to a different queue.
		// Only applies when a queue and service have been specified as a destination,
		// allowing management through infra, such as expoential retry back-off.
		if areRetriesExhausted(maxAttempts, r) {
			if len(problematicQueue) > 0 && len(problematicService) > 0 {
				log.Printf("Pushing problematic work to %s\n", problematicQueue)
				// Issue a task to pass the message to the next queue
				createTask("moraisworkrunner", location, problematicQueue, problematicService, string(body))
				return
			}
			// Issue a task of "failure" to the notifier queue
			body, err = proto.Marshal(&workResponse)
			if err != nil {
				log.Printf("Failed to Marshal work response proto: %v\n", body)
				return
			}
			createTask("moraisworkrunner", location, target, webhookURL, string(body))
		}
		return
	}
	// Work processing succeeded
	w.WriteHeader(http.StatusAccepted)
	// Issue a task to the notifier queue once handling has completed
	body, err = proto.Marshal(&workResponse)
	if err != nil {
		log.Printf("Failed to Marshal work response proto: %v\n", body)
		return
	}
	createTask("moraisworkrunner", location, target, webhookURL, string(body))
}

func processWork(in *work_messages.SvcWorkRequest) error {
	// do "work" with the request
	log.Printf("%s - %s -> %s\n", in.FileMetadata.GetMd5(), in.SourceFile, in.WebhookUrl)
	if in.SourceFile == "invalid" {
		return fmt.Errorf("Error: invalid source file")
	}
	return nil
}

func areRetriesExhausted(maxRetryCount int64, req *http.Request) bool {
	retries := req.Header.Get(retryCountHeader)
	if retries != "" {
		r, err := strconv.ParseInt(retries, 10, 32)
		if err != nil {
			log.Printf("Failed to parse header value %s to an int: %v", retries, err)
			return false
		}
		return r >= maxRetryCount
	}
	return false
}

func main() {
	log.Print("starting server...\n")
	http.HandleFunc("/", handler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on port %s\n", port)
	log.Print(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
	os.Exit(0)
}
