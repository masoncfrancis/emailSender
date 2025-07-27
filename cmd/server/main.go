package main

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

// WebhookPayload represents the expected structure of the incoming JSON from PowerShell
type WebhookPayload struct {
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
	Source       string `json:"source"`
	Destination  string `json:"destination"`
	ExitCode     int    `json:"exitCode"`
	EmailContent string `json:"emailContent"` // This field holds the pre-formatted email body
}

// sendEmail sends an email using the configured SMTP server.
func sendEmail(subject, body string) error {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file, attempting to use system environment variables: %v", err)
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	senderEmail := os.Getenv("SENDER_EMAIL")
	recipientEmail := os.Getenv("RECIPIENT_EMAIL")

	// Basic validation for environment variables
	if smtpHost == "" || smtpPort == "" || smtpUsername == "" || smtpPassword == "" || senderEmail == "" || recipientEmail == "" {
		return fmt.Errorf("SMTP configuration missing in .env or environment variables. Please check SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, SENDER_EMAIL, RECIPIENT_EMAIL")
	}

	// Authentication
	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)

	// Construct the full email message
	msg := []byte("From: " + senderEmail + "\r\n" +
		"To: " + recipientEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\r\n" + // Ensure plain text and UTF-8
		"\r\n" +
		body)

	// Send the email
	addr := smtpHost + ":" + smtpPort
	log.Printf("Attempting to send email from %s to %s via %s...", senderEmail, recipientEmail, addr)
	err = smtp.SendMail(addr, auth, senderEmail, []string{recipientEmail}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Println("Email sent successfully!")
	return nil
}

func main() {
	// Initialize Fiber app
	app := fiber.New()

	// Define the webhook endpoint
	app.Post("/webhook/robocopy-failure", func(c *fiber.Ctx) error {
		// Parse the incoming JSON payload
		payload := new(WebhookPayload)
		if err := c.BodyParser(payload); err != nil {
			log.Printf("Error parsing JSON body: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot parse request body",
			})
		}

		log.Printf("Received webhook for Robocopy status: %s, Exit Code: %d", payload.Status, payload.ExitCode)
		log.Printf("Email content length: %d bytes", len(payload.EmailContent))

		// Extract subject from the email content (first line after "Subject: ")
		// The PowerShell script formats the subject as "Subject: Robocopy Failure Notification"
		// We'll look for this line to extract the actual subject for the email.
		emailLines := strings.Split(payload.EmailContent, "\n")
		subject := "Robocopy Notification" // Default subject
		for _, line := range emailLines {
			if strings.HasPrefix(line, "Subject:") {
				subject = strings.TrimSpace(strings.TrimPrefix(line, "Subject:"))
				break
			}
		}

		// Send the email with the extracted content
		if err := sendEmail(subject, payload.EmailContent); err != nil {
			log.Printf("Error sending email: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to send email notification",
				"details": err.Error(),
			})
		}

		// Return success response
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Webhook received and email sent successfully",
		})
	})

	// Start the Fiber server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // Default port if not specified in .env
	}
	log.Printf("Fiber listening on :%s", port)
	log.Fatal(app.Listen(":" + port))
}
