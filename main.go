package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"

	"github.com/joho/godotenv"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

type payload struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

func main() {
	pathPtr := flag.String("file", "", "The path to your selected file")
	voicePtr := flag.String("voice", "alloy", "The voice your text will be read in. Choices are: - alloy (default)\n")

	flag.Parse()

	text, err := os.ReadFile(*pathPtr)
	if err != nil {
		log.Fatalf("Could not read file %q: %v", *pathPtr, err)
	}

	postToAPI(string(text), *voicePtr)

	if _, err := os.Stat("response.mp3"); err == nil {
		sendEmail()
	}
}

func sendEmail() {
	smtpServer := "smtp.example.com"
	smtpPort := "587"
	sender := ""
	password := ""
	recipient := ""

	auth := smtp.PlainAuth("", sender, password, smtpServer)

	msg := []byte("To: " + recipient + "\r\n" +
		"Subject: Your audiofile\r\n" +
		"\r\n" +
		"Here's a some text")

	err := smtp.SendMail(smtpServer+":"+smtpPort, auth, sender, []string{recipient}, msg)
	if err != nil {
		log.Fatalf("Could not send email: %v", err)
	}

	fmt.Println("Email sent")
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

		if pageNum > 3 {
			break
		}

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

func postToAPI(content, voice string) {

	url := "https://api.openai.com/v1/audio/speech"

	payload := payload{
		Model: "tts-1",
		Input: content,
		Voice: voice,
	}

	p, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Could not marshall payload %v: %v", payload, err)
	}

	// payloadObject := []byte(`
	//        {
	//            "model": "tts-1",
	//            "input": "The quick brown fox jumped over the lazy dog",
	//            "voice": "alloy"
	//        }
	//    `)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(p))
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
