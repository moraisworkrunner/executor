package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	work_messages "github.com/moraisworkrunner/work-messages"
	"google.golang.org/protobuf/proto"
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

	// Parse the body and protobuf message from the request
	in := &work_messages.SvcWorkRequest{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		workResponse.Error = &work_messages.Error{
			Message: fmt.Sprintf("Failed to read request"),
		}
		fmt.Printf("Failed to read work request: %v\n", r.Body)
		// Do not retry
		return
	}
	if err := proto.Unmarshal(body, in); err != nil {
		workResponse.Error = &work_messages.Error{
			Message: fmt.Sprintf("Failed to parse work request: %s", err),
		}
		fmt.Printf("Failed to unmarshal proto: %v\n", body)
		// Do not retry
		return
	}
	workResponse.Context = in.Context
	webhookURL := in.WebhookUrl

	// Issue a task to the notifier queue once handling has completed
	defer func(r *work_messages.SvcWorkResponse) {
		body, err := proto.Marshal(r)
		if err != nil {
			fmt.Printf("Failed to Marshal work response proto: %v\n", body)
			return
		}
		createTask("moraisworkrunner", location, target, string(body), webhookURL)
	}(&workResponse)

	// Do something with the payload, and return the appropriate status
	if err := processWork(in); err != nil {
		// process the work request
		workResponse.Error = &work_messages.Error{
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
}

func processWork(in *work_messages.SvcWorkRequest) error {
	// do "work" with the request
	fmt.Printf("%s - %s -> %s\n", in.FileMetadata.GetMd5(), in.SourceFile, in.WebhookUrl)
	if in.SourceFile == "invalid" {
		return fmt.Errorf("Error: invalid source file")
	}
	return nil
}

func main() {
	fmt.Print("starting server...\n")
	http.HandleFunc("/", handler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("listening on port %s\n", port)
	fmt.Print(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
	os.Exit(0)
}
