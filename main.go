package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	url := "https://api.openai.com/v1/audio/speech"

	payload := []byte(`
        {
            "model": "tts-1",
            "input": "The quick brown fox jumped over the lazy dog",
            "voice": "alloy"
        }
    `)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		log.Fatalf("Could not create POST request: %v", err)
	}

	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	apiKey := "Bearer " + os.Getenv("OPENAI_API_KEY")
	req.Header.Add("Authorization", apiKey)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Could not make POST request: %v", err)
	}
	defer resp.Body.Close()

	file, err := os.Create("response.mp3")
	if err != nil {
		log.Fatalf("Could not create MP3 file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
}
