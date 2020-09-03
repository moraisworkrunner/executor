package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"cloud.google.com/go/logging"
	work_messages "github.com/moraisworkrunner/work-messages"
	"google.golang.org/protobuf/proto"
)

var (
	logger *log.Logger
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
	maxAttempts, err := strconv.Atoi(maxAttemptsStr)
	if err != nil {
		maxAttempts = 20
	}
	// Where to send, and who will service, problematic requests
	problematicQueue := os.Getenv("PROBLEM_QUEUE")
	problematicService := os.Getenv("PROBLEM_SERVICE")

	// Parse the body and protobuf message from the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		workResponse.Error = &work_messages.Error{
			Message: fmt.Sprintf("Failed to read request"),
		}
		logger.Printf("Failed to read work request: %v\n", r.Body)
		// Do not retry
		return
	}
	in := &work_messages.SvcWorkRequest{}
	if err := proto.Unmarshal(body, in); err != nil {
		workResponse.Error = &work_messages.Error{
			Message: fmt.Sprintf("Failed to parse work request: %s", err),
		}
		logger.Printf("Failed to unmarshal proto: %v\n", body)
		// Do not retry
		return
	}
	workResponse.Context = in.Context
	webhookURL := in.WebhookUrl

	taskExecutionCount, err := strconv.Atoi(r.Header.Get("X-CloudTasks-TaskExecutionCount"))
	if err != nil {
		logger.Printf("Warning: Cannot read X-CloudTasks-TaskExecutionCount header in request\n")
	}
	logger.Printf("Task-Execution-Count: %d\n", taskExecutionCount)
	logger.Printf("")
	// Do something with the payload, and return the appropriate status
	if err := processWork(in); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		workResponse.Error = &work_messages.Error{
			Message: err.Error(),
		}
		// "maxAttempts" may be leveraged to push problematic work to a different queue.
		// Only applies when a queue and service have been specified as a destination,
		// allowing management through infra, such as expoential retry back-off.
		if taskExecutionCount == maxAttempts {
			logger.Printf("Max-Attempts reached: %d\n", taskExecutionCount)
			if len(problematicQueue) > 0 && len(problematicService) > 0 {
				logger.Printf("Pushing problematic work to %s\n", problematicQueue)
				// Issue a task to pass the message to the next queue
				createTask("moraisworkrunner", location, problematicQueue, problematicService, string(body))
				return
			}
			// Issue a task of "failure" to the notifier queue
			body, err = proto.Marshal(&workResponse)
			if err != nil {
				logger.Printf("Failed to Marshal work response proto: %v\n", body)
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
		logger.Printf("Failed to Marshal work response proto: %v\n", body)
		return
	}
	createTask("moraisworkrunner", location, target, webhookURL, string(body))
}

func processWork(in *work_messages.SvcWorkRequest) error {
	// do "work" with the request
	logger.Printf("%s - %s -> %s\n", in.FileMetadata.GetMd5(), in.SourceFile, in.WebhookUrl)
	if in.SourceFile == "invalid" {
		return fmt.Errorf("Error: invalid source file")
	}
	return nil
}

func main() {
	// Creates a client.
	client, err := logging.NewClient(context.Background(), "moraisworkrunner")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	logger = client.Logger("morais").StandardLogger(logging.Info)

	logger.Print("starting server...\n")
	http.HandleFunc("/", handler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Printf("listening on port %s\n", port)
	logger.Print(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
	os.Exit(0)
}
