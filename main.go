package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

type Sender struct {
	Auth    smtp.Auth
	Details mail.Address
}

type Email struct {
	From        string
	To          string
	Subject     string
	Body        string
	Attachments map[string][]byte
}

type ApiPayload struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

func NewSender() *Sender {
	smtpServer := os.Getenv("APP_EMAIL_SERVER")
	sender := os.Getenv("APP_EMAIL_USERNAME")
	password := os.Getenv("APP_EMAIL_PASSWORD")

	auth := smtp.PlainAuth("", sender, password, smtpServer)

	return &Sender{
		Auth: auth,
		Details: mail.Address{
			Name:    os.Getenv("APP_NAME"),
			Address: os.Getenv("APP_EMAIL_USERNAME"),
		},
	}
}

func (s *Sender) NewEmail(to, subject, body string) *Email {
	return &Email{
		From:        s.Details.Address,
		To:          to,
		Subject:     subject,
		Body:        body,
		Attachments: make(map[string][]byte),
	}
}

func (s *Sender) Send(e *Email) error {
	addr := fmt.Sprintf("%s:%s", os.Getenv("APP_EMAIL_SERVER"), os.Getenv("APP_EMAIL_PORT"))
	msg := e.ToBytes()

	err := smtp.SendMail(addr, s.Auth, s.Details.Name, []string{e.To}, msg)
	if err != nil {
		return err
	}

	fmt.Println("Email sent")
	return nil
}

func (e *Email) AttachFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	_, name := filepath.Split(path)
	e.Attachments[name] = b
	return nil
}

func (e *Email) ToBytes() []byte {
	buf := bytes.NewBuffer(nil)
	hasAttachments := len(e.Attachments) > 0

	buf.WriteString(fmt.Sprintf("To: %s\n", e.To))
	buf.WriteString(fmt.Sprintf("Subject: %s\n", e.Subject))
	buf.WriteString("MIME-Version: 1.0\n")

	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	if hasAttachments {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\n", boundary))
		buf.WriteString(fmt.Sprintf("--%s\n", boundary))
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\n\n")
	}

	buf.WriteString(e.Body)

	if hasAttachments {
		for k, v := range e.Attachments {
			buf.WriteString(fmt.Sprintf("\n\n--%s\n", boundary))
			buf.WriteString("Content-Type: audio/mpeg\n")
			buf.WriteString("Content-Transfer-Encoding: base64\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\n\n", k))

			b := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
			base64.StdEncoding.Encode(b, v)
			buf.Write(b)
		}

		buf.WriteString(fmt.Sprintf("\n--%s--\n", boundary))
	}

	return buf.Bytes()
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	pathPtr := flag.String("file", "", "The path to your selected file")
	voicePtr := flag.String("voice", "alloy", "The voice your text will be read in. Choices are: - alloy (default)\n")
	_ = voicePtr

	flag.Parse()

	// text, err := os.ReadFile(*pathPtr)
	if err != nil {
		log.Fatalf("Could not read file %q: %v", *pathPtr, err)
	}

	// postToAPI(string(text), *voicePtr)

	if _, err := os.Stat("response.mp3"); err == nil {
		sender := NewSender()
		email := sender.NewEmail(os.Getenv("APP_EMAIL_USERNAME"), "You Audio File", "Here it is - Great job!")
		email.AttachFile("response.mp3")
		sender.Send(email)
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

	payload := ApiPayload{
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
