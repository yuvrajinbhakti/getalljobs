package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// Platform represents a job platform with its search configuration
type Platform struct {
	Name      string
	BaseURL   string
	QueryPath string
	Selector  PlatformSelector
}

// PlatformSelector defines the CSS selectors for extracting job details
type PlatformSelector struct {
	JobContainer string
	Title        string
	Company      string
	Location     string
	Description  string
	Salary       string
	PostedDate   string
	NextPage     string // Added for pagination support
}

// Job represents a detailed job listing
type Job struct {
	Platform    string
	Title       string
	Company     string
	Location    string
	Description string
	Salary      string
	PostedDate  string
	URL         string
}

// NotificationConfig stores notification settings
type NotificationConfig struct {
	Email    EmailConfig
	WhatsApp WhatsAppConfig
}

// EmailConfig holds email notification details
type EmailConfig struct {
	Enabled  bool
	From     string
	Password string
	To       string
	SMTPHost string
	SMTPPort int
}

// WhatsAppConfig holds WhatsApp notification details
type WhatsAppConfig struct {
	Enabled  bool
	Number   string
	Provider string
}

// JobScraper represents the core job scraping functionality
type JobScraper struct {
	collector   *colly.Collector
	rateLimiter *time.Ticker
	jobs        []Job
	jobsMutex   sync.Mutex
}

// Helper function to create a collector
func createCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(3),
		colly.UserAgent(randomUserAgent()),
		colly.AllowURLRevisit(),
	)

	c.WithTransport(&http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	})

	return c
}

// Helper function to generate random user agent
func randomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

// Sanitize text by trimming whitespace
func sanitizeText(text string) string {
	return strings.TrimSpace(text)
}

// NewJobScraper creates a new job scraper instance
func NewJobScraper() *JobScraper {
	return &JobScraper{
		collector:   createCollector(),
		rateLimiter: time.NewTicker(time.Second * 2), // Rate limit requests
		jobs:        []Job{},
	}
}

// Scrape performs job scraping for a specific platform
func (js *JobScraper) Scrape(platform Platform, jobTitle, location string) error {
	// Construct search URL
	baseURL := fmt.Sprintf("%s%s?q=%s&l=%s",
		platform.BaseURL,
		platform.QueryPath,
		url.QueryEscape(jobTitle),
		url.QueryEscape(location),
	)

	// Setup collector error handling
	js.collector.OnError(func(r *colly.Response, err error) {
		log.Printf("Scrape Error on %s: %v", platform.Name, err)
	})

	// Parse job listings
	js.collector.OnHTML(platform.Selector.JobContainer, func(e *colly.HTMLElement) {
		job := Job{
			Platform:    platform.Name,
			Title:       sanitizeText(e.ChildText(platform.Selector.Title)),
			Company:     sanitizeText(e.ChildText(platform.Selector.Company)),
			Location:    sanitizeText(e.ChildText(platform.Selector.Location)),
			Description: sanitizeText(e.ChildText(platform.Selector.Description)),
			Salary:      sanitizeText(e.ChildText(platform.Selector.Salary)),
			PostedDate:  sanitizeText(e.ChildText(platform.Selector.PostedDate)),
			URL:         e.Request.URL.String(),
		}

		js.jobsMutex.Lock()
		js.jobs = append(js.jobs, job)
		js.jobsMutex.Unlock()
	})

	// Visit the URL
	return js.collector.Visit(baseURL)
}

// SendEmailNotification sends job listings via email
func SendEmailNotification(config EmailConfig, jobs []Job) error {
	if len(jobs) == 0 {
		return fmt.Errorf("no jobs to send")
	}

	// Compose email body
	var emailBody strings.Builder
	emailBody.WriteString("Daily Job Listings:\n\n")
	for _, job := range jobs {
		jobDetails := fmt.Sprintf(
			"Title: %s\nCompany: %s\nLocation: %s\nURL: %s\n\n",
			job.Title, job.Company, job.Location, job.URL,
		)
		emailBody.WriteString(jobDetails)
	}

	// Email authentication and sending
	auth := smtp.PlainAuth("", config.From, config.Password, config.SMTPHost)

	msg := []byte(
		"To: " + config.To + "\r\n" +
			"Subject: Daily Job Listings\r\n" +
			"\r\n" +
			emailBody.String(),
	)

	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort),
		auth,
		config.From,
		[]string{config.To},
		msg,
	)

	return err
}

// SendWhatsAppNotification sends job listings via WhatsApp (placeholder)
func SendWhatsAppNotification(config WhatsAppConfig, jobs []Job) error {
	// Implement WhatsApp notification logic
	log.Println("WhatsApp notification not implemented")
	return nil
}

// SaveToCSV saves job listings to a CSV file
func SaveToCSV(filename string, jobs []Job) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV headers
	headers := []string{"Platform", "Title", "Company", "Location", "Description", "Salary", "PostedDate", "URL"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("error writing headers: %v", err)
	}

	// Write job data
	for _, job := range jobs {
		record := []string{
			job.Platform,
			job.Title,
			job.Company,
			job.Location,
			job.Description,
			job.Salary,
			job.PostedDate,
			job.URL,
		}
		if err := writer.Write(record); err != nil {
			log.Printf("Error writing job record: %v", err)
		}
	}

	return nil
}

func main() {
	// Command-line flags
	jobTitle := flag.String("title", "software engineer", "Job title to search")
	location := flag.String("location", "remote", "Job location")
	outputFile := flag.String("output", "daily_jobs.csv", "Output CSV file")

	// Email notification flags
	emailEnabled := flag.Bool("email-enabled", false, "Enable email notifications")
	emailFrom := flag.String("email-from", "", "Sender email")
	emailPassword := flag.String("email-pass", "", "Email password")
	emailTo := flag.String("email-to", "", "Recipient email")
	smtpHost := flag.String("smtp-host", "smtp.gmail.com", "SMTP host")
	smtpPort := flag.Int("smtp-port", 587, "SMTP port")

	// WhatsApp notification flags
	whatsappEnabled := flag.Bool("whatsapp-enabled", false, "Enable WhatsApp notifications")
	whatsappNumber := flag.String("whatsapp-number", "", "WhatsApp number")
	whatsappProvider := flag.String("whatsapp-provider", "", "WhatsApp provider")

	flag.Parse()

	// Create notification configuration
	notificationConfig := NotificationConfig{
		Email: EmailConfig{
			Enabled:  *emailEnabled,
			From:     *emailFrom,
			Password: *emailPassword,
			To:       *emailTo,
			SMTPHost: *smtpHost,
			SMTPPort: *smtpPort,
		},
		WhatsApp: WhatsAppConfig{
			Enabled:  *whatsappEnabled,
			Number:   *whatsappNumber,
			Provider: *whatsappProvider,
		},
	}

	// Define platforms to scrape
	platforms := []Platform{
		{
			Name:      "Indeed",
			BaseURL:   "https://www.indeed.com",
			QueryPath: "/jobs",
			Selector: PlatformSelector{
				JobContainer: ".job_seen_beacon",
				Title:        ".jobTitle",
				Company:      ".companyName",
				Location:     ".companyLocation",
				Description:  ".job-snippet",
				Salary:       ".salary-snippet",
				PostedDate:   ".date",
			},
		},
		// Add more platforms as needed
	}

	// Create job scraper
	scraper := NewJobScraper()

	// Scrape jobs from different platforms
	var allJobs []Job
	var wg sync.WaitGroup
	var jobsMutex sync.Mutex

	for _, platform := range platforms {
		wg.Add(1)
		go func(p Platform) {
			defer wg.Done()
			err := scraper.Scrape(p, *jobTitle, *location)
			if err != nil {
				log.Printf("Scraping error for %s: %v", p.Name, err)
				return
			}

			jobsMutex.Lock()
			allJobs = append(allJobs, scraper.jobs...)
			jobsMutex.Unlock()
		}(platform)
	}
	wg.Wait()

	// Save to CSV
	if err := SaveToCSV(*outputFile, allJobs); err != nil {
		log.Printf("CSV saving error: %v", err)
	}

	// Send notifications if jobs are found
	if len(allJobs) > 0 {
		// Email notification
		if notificationConfig.Email.Enabled {
			if err := SendEmailNotification(notificationConfig.Email, allJobs); err != nil {
				log.Printf("Email notification failed: %v", err)
			}
		}

		// WhatsApp notification
		if notificationConfig.WhatsApp.Enabled {
			if err := SendWhatsAppNotification(notificationConfig.WhatsApp, allJobs); err != nil {
				log.Printf("WhatsApp notification failed: %v", err)
			}
		}
	}
}


/*
go run job_scraper.go \
  -title="software engineer" \
  -location="remote" \
  -email-enabled=true \
  -email-from=your@email.com \
  -email-pass=yourpassword \
  -email-to=recipient@email.com \
  -output=daily_jobs.csv
*/