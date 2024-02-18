package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

func main() {
    file := ""

    if err := outputPdfText(file); err != nil {
        fmt.Printf("Could not output PDF: %v", err)
    }
}

func outputPdfText(inputPath string) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	defer f.Close()

	pdfReader, err := model.NewPdfReader(f)
	if err != nil {
		return err
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return err
	}

	fmt.Printf("--------------------\n")
	fmt.Printf("PDF to text extraction:\n")
	fmt.Printf("--------------------\n")
	for i := 0; i < numPages; i++ {
		pageNum := i + 1

		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			return err
		}

		ex, err := extractor.New(page)
		if err != nil {
			return err
		}

		text, err := ex.ExtractText()
		if err != nil {
			return err
		}

		fmt.Println("------------------------------")
		fmt.Printf("Page %d:\n", pageNum)
		fmt.Printf("\"%s\"\n", text)
		fmt.Println("------------------------------")
	}

	return nil
}

func postToAPI() {

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
