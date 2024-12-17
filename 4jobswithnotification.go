package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// Existing struct definitions remain the same as in previous code
// (Platform, PlatformSelector, Job, NotificationConfig, JobFilters)

// JobScraper manages the scraping process across multiple platforms
// Job represents a job listing
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

// PlatformSelector defines the CSS selectors for extracting job details
type PlatformSelector struct {
	JobContainer string
	Title        string
	Company      string
	Location     string
	Description string
	Salary       string
	PostedDate   string
}

// Platform represents a job platform with its search configuration
type Platform struct {
	Name      string
	BaseURL   string
	QueryPath string
	Selector  PlatformSelector
	Filters   map[string]string
}

// NotificationConfig stores configuration for email and WhatsApp notifications
type NotificationConfig struct {
	EmailEnabled     bool
	WhatsAppEnabled  bool
	EmailFrom        string
	EmailPassword    string
	EmailTo          string
	SMTPHost         string
	SMTPPort         int
	WhatsAppNumber   string
	WhatsAppProvider string
}

// JobFilters defines criteria for filtering job listings
type JobFilters struct {
	MaxExperience int
	Location      string
	JobType       string
	Keyword       string
}
// NewExtendedJobScraper creates a new extended job scraper with notification support
func NewExtendedJobScraper(platforms []Platform, notificationConfig *NotificationConfig, filters JobFilters) *ExtendedJobScraper {
	// Create base job scraper
	baseScraper := NewJobScraper(platforms)

	// Create extended scraper
	return &ExtendedJobScraper{
		JobScraper:         *baseScraper,
		NotificationConfig: notificationConfig,
		Filters:            filters,
	}
}

// Scrape method (add this method to JobScraper)
func (js *JobScraper) Scrape(platform Platform, jobTitle, location string, filters map[string]string) {
	// Wait for rate limiter to avoid overwhelming the target website
	err := js.rateLimiter.Wait(context.Background())
	if err != nil {
		log.Printf("Rate limit error: %v", err)
		return
	}

	// Reset collector for each platform
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

		// Only add job if it has essential information
		if job.Title != "" && job.Company != "" {
			js.jobsMutex.Lock()
			js.jobs = append(js.jobs, job)
			js.jobsMutex.Unlock()
		}
	})

	// Construct search URL
	baseURL := fmt.Sprintf("%s%s?q=%s&l=%s",
		platform.BaseURL,
		platform.QueryPath,
		url.QueryEscape(jobTitle),
		url.QueryEscape(location),
	)

	// Add additional filters
	for key, value := range filters {
		baseURL += fmt.Sprintf("&%s=%s", key, url.QueryEscape(value))
	}

	// Visit the constructed URL
	err = js.collector.Visit(baseURL)
	if err != nil {
		log.Printf("Failed to visit URL for %s: %v", platform.Name, err)
	}

	// Wait for all requests to complete
	js.collector.Wait()
}

// FilterJobs method for ExtendedJobScraper
func (ejs *ExtendedJobScraper) FilterJobs() []Job {
	var filteredJobs []Job

	for _, job := range ejs.jobs {
		if ejs.matchesFilters(job) {
			filteredJobs = append(filteredJobs, job)
		}
	}

	return filteredJobs
}

// matchesFilters checks if a job matches the specified criteria
func (ejs *ExtendedJobScraper) matchesFilters(job Job) bool {
	// Basic filtering logic - customize as needed
	locationMatch := strings.Contains(
		strings.ToLower(job.Location), 
		strings.ToLower(ejs.Filters.Location),
	)
	
	isRemote := strings.Contains(
		strings.ToLower(job.Location), 
		"remote",
	)
	
	isFresherJob := strings.Contains(
		strings.ToLower(job.Title), 
		"fresher",
	) || strings.Contains(
		strings.ToLower(job.Title), 
		"entry level",
	)

	return (locationMatch || isRemote) && isFresherJob
}

// SaveToCSV method (add this method to JobScraper)
func (js *JobScraper) SaveToCSV(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV headers
	headers := []string{"Platform", "Title", "Company", "Location", "Description", "Salary", "PostedDate", "URL"}
	if err := writer.Write(headers); err != nil {
		log.Fatalf("Error writing headers: %v", err)
	}

	// Write job data
	js.jobsMutex.Lock()
	defer js.jobsMutex.Unlock()

	for _, job := range js.jobs {
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

	log.Printf("Saved %d jobs to %s", len(js.jobs), filename)
}

// Helper functions for collector and other utilities
func randomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

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

func sanitizeText(text string) string {
	return strings.TrimSpace(text)
}

// Existing email and WhatsApp notification methods remain the same as in previous code
// ... (SendEmailNotification and SendWhatsAppNotification methods)

// Main function remains the same as in previous code
func main() {
	// (Keep the existing main function from the previous code)
	// Command-line flags for job search
	jobTitle := flag.String("title", "software engineer", "Job title to search for")
	location := flag.String("location", "remote", "Job location to search for")
	outputFile := flag.String("output", "jobswithnotification.csv", "Output file for job results")

	// Email notification flags
	emailEnabled := flag.Bool("email-enabled", false, "Enable email notifications")
	emailFrom := flag.String("email-from", "", "Sender email address")
	emailPassword := flag.String("email-pass", "", "Email password or app password")
	emailTo := flag.String("email-to", "", "Recipient email address")
	smtpHost := flag.String("smtp-host", "smtp.gmail.com", "SMTP host server")
	smtpPort := flag.Int("smtp-port", 587, "SMTP port")

	// WhatsApp notification flags
	whatsappEnabled := flag.Bool("whatsapp-enabled", false, "Enable WhatsApp notifications")
	whatsappNumber := flag.String("whatsapp-number", "", "WhatsApp number for notifications")
	whatsappProvider := flag.String("whatsapp-provider", "", "WhatsApp notification provider")

	// Parse command-line flags
	flag.Parse()

	// Create notification configuration
	notificationConfig := &NotificationConfig{
		EmailEnabled:     *emailEnabled,
		WhatsAppEnabled:  *whatsappEnabled,
		EmailFrom:        *emailFrom,
		EmailPassword:    *emailPassword,
		EmailTo:          *emailTo,
		SMTPHost:         *smtpHost,
		SMTPPort:         *smtpPort,
		WhatsAppNumber:   *whatsappNumber,
		WhatsAppProvider: *whatsappProvider,
	}

	// Job filters
	jobFilters := JobFilters{
		MaxExperience: 1, 
		Location:      "India",
		JobType:       "Remote",
		Keyword:       "Fresher",
	}

	// Job platforms
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
		// Add other platforms as before
	}

	// Create extended scraper
	scraper := NewExtendedJobScraper(platforms, notificationConfig, jobFilters)

	var wg sync.WaitGroup
	for _, platform := range platforms {
		wg.Add(1)
		go func(p Platform) {
			defer wg.Done()
			scraper.Scrape(p, *jobTitle, *location, p.Filters)
		}(platform)
	}
	wg.Wait()

	// Filter jobs
	filteredJobs := scraper.FilterJobs()

	// Send notifications if jobs are found
	if len(filteredJobs) > 0 {
		// Email notification
		if notificationConfig.EmailEnabled {
			err := scraper.SendEmailNotification(filteredJobs)
			if err != nil {
				log.Printf("Email notification failed: %v", err)
			}
		}

		// WhatsApp notification
		if notificationConfig.WhatsAppEnabled {
			err := scraper.SendWhatsAppNotification(filteredJobs)
			if err != nil {
				log.Printf("WhatsApp notification failed: %v", err)
			}
		}

		// Save to CSV
		scraper.SaveToCSV(*outputFile)
	}
}



/*
go run 4jobswithnotification.go \
  -email-enabled=true \
  -email-from=yuvrajsinghnain03@gmail.com \
  -email-pass=your-app-password \
  -email-to=yuvrajsinghnain03@gmail.com \
  -title="software engineer" \
  -location="remote"

*/