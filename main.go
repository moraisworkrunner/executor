package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	work_messages "github.com/moraisworkrunner/work-messages"
	"google.golang.org/protobuf/proto"
)

func handler(_ http.ResponseWriter, r *http.Request) {
	workResponse := work_messages.SvcWorkResponse{}

	target := os.Getenv("NOTIFIER_QUEUE")
	if target == "" {
		target = "queue"
	}
	location := os.Getenv("NOTIFIER_LOCATION")
	if location == "" {
		location = "nowhere"
	}

	log.Print("received a request")

	// Issue a task to the notifier queue once handling has completed
	defer func(r *work_messages.SvcWorkResponse) {
		body, err := proto.Marshal(r)
		if err != nil {

		}
		createTask("moraisworkrunner", location, target, string(body))
	}(&workResponse)

	// Parse the body and protobuf message from the request
	in := &work_messages.SvcWorkRequest{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		workResponse.Error = &work_messages.Error{
			Message: fmt.Sprintf("Failed to read request"),
		}
		return
	}
	if err := proto.Unmarshal(body, in); err != nil {
		workResponse.Error = &work_messages.Error{
			Message: fmt.Sprintf("Failed to parse work request: %s", err),
		}
		return
	}
	workResponse.Context = in.Context

	// process the work request
	processWork(in)
}

func processWork(in *work_messages.SvcWorkRequest) {
	// do "work" with the request
	fmt.Printf("%s - %s", in.FileMetadata.GetMd5(), in.SourceFile)

}

func main() {
	log.Print("starting server...")
	http.HandleFunc("/", handler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
