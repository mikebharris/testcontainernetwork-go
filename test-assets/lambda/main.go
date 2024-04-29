package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	lambda.Start(func() error {
		resp, err := http.Get(os.Getenv("API_ENDPOINT"))
		if resp.StatusCode != http.StatusOK {
			log.Printf("expected status %v, got status %v", http.StatusOK, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("reading respons body: %v", err)
		}

		var returnedJson struct {
			Message string `json:"message"`
		}
		if err = json.Unmarshal(body, &returnedJson); err != nil {
			log.Printf("unmarshalling %s: %v", body, err)
		}

		log.Printf("Wiremock returned a message of %s", returnedJson.Message)
		return nil
	})
}
