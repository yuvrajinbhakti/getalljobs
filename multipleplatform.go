package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	// "net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/time/rate"
)

// Platform represents a job platform with specific configurations
type Platform struct {
	Name      string
	BaseURL   string
	QueryPath string
	Filters   map[string]string
	Selector  PlatformSelector
}

// PlatformSelector contains CSS selectors for job details
type PlatformSelector struct {
	JobContainer string
	Title        string
	Company      string
	Location     string
	Description string
	Salary       string
	PostedDate   string
}

// Job represents a job listing with comprehensive details
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

// JobScraper handles job scraping from multiple platforms
type JobScraper struct {
	jobs        []Job
	jobsMutex   sync.Mutex
	rateLimiter *rate.Limiter
	platforms   []Platform
}

// NewJobScraper initializes the scraper with advanced configurations
func NewJobScraper(platforms []Platform) *JobScraper {
	// Implement exponential backoff for rate limiting
	rateLimiter := rate.NewLimiter(rate.Every(1*time.Second), 1) // More conservative rate limiting

	return &JobScraper{
		jobs:        []Job{},
		rateLimiter: rateLimiter,
		platforms:   platforms,
	}
}

// randomUserAgent generates a more comprehensive list of user agents
func randomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:89.0) Gecko/20100101 Firefox/89.0",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

// createCollector creates a new collector with advanced anti-detection techniques
func createCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(3),
		colly.UserAgent(randomUserAgent()),
		colly.AllowURLRevisit(),
	)

	// Configure browser-like headers
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "none")
		r.Headers.Set("Sec-Fetch-User", "?1")
	})

	return c
}

// Scrape scrapes jobs from a specific platform with improved error handling
func (js *JobScraper) Scrape(platform Platform, jobTitle, location string, filters map[string]string) {
	// Wait for rate limiter
	err := js.rateLimiter.Wait(context.Background())
	if err != nil {
		log.Printf("Rate limit error: %v", err)
		return
	}

	// Create a new collector for each platform
	collector := createCollector()

	// Error handling and logging
	collector.OnError(func(r *colly.Response, err error) {
		log.Printf("Scrape Error for %s: %v, Status Code: %d, URL: %s", 
			platform.Name, err, r.StatusCode, r.Request.URL)
	})

	// Job extraction
	collector.OnHTML(platform.Selector.JobContainer, func(e *colly.HTMLElement) {
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
		
		// Only add non-empty jobs
		if job.Title != "" && job.Company != "" {
			js.jobsMutex.Lock()
			js.jobs = append(js.jobs, job)
			js.jobsMutex.Unlock()
		}
	})

	// Construct URL with all parameters
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

	// Multiple attempts to visit the URL
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := collector.Visit(baseURL)
		if err == nil {
			break
		}
		log.Printf("Attempt %d failed: %v", attempt+1, err)
		time.Sleep(time.Duration(attempt+1) * 3 * time.Second)
	}

	// Wait for all requests to complete
	collector.Wait()
}

// sanitizeText cleans up and trims text
func sanitizeText(text string) string {
	return strings.TrimSpace(text)
}

// SaveToCSV saves the scraped jobs to a CSV file
func (js *JobScraper) SaveToCSV(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"Platform", "Title", "Company", "Location", "Description", "Salary", "Posted Date", "URL"}
	writer.Write(headers)

	for _, job := range js.jobs {
		record := []string{job.Platform, job.Title, job.Company, job.Location, job.Description, job.Salary, job.PostedDate, job.URL}
		writer.Write(record)
	}
}

// SaveToJSON saves the scraped jobs to a JSON file
func (js *JobScraper) SaveToJSON(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	jsonData, err := json.MarshalIndent(js.jobs, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal data to JSON: %v", err)
	}

	file.Write(jsonData)
}

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Command-line inputs
	jobTitle := flag.String("jobTitle", "Software Engineer", "Job title to search")
	location := flag.String("location", "Remote", "Location to search jobs")
	output := flag.String("output", "multipleplatformjobs.csv", "Output file name (CSV/JSON)")
	flag.Parse()

	// Platforms to scrape with more specific selectors
	platforms := []Platform{
		{
			Name:      "Indeed", 
			BaseURL:   "https://www.indeed.com", 
			QueryPath: "/jobs", 
			Filters:   map[string]string{},
			Selector: PlatformSelector{
				JobContainer: "div[class^='job_seen_beacon']",
				Title:        "h2.jobTitle span[title]",
				Company:      "span.companyName",
				Location:     "div.companyLocation",
				Description: "div.job-snippet",
				Salary:       "div.metadata.salary-snippet-container",
				PostedDate:   "span.date",
			},
		},
		{
			Name:      "LinkedIn", 
			BaseURL:   "https://www.linkedin.com", 
			QueryPath: "/jobs/search", 
			Filters:   map[string]string{"f_TPR": "r2592000"}, // last 30 days filter
			Selector: PlatformSelector{
				JobContainer: "div.base-card",
				Title:        "h3.base-search-card__title",
				Company:      "h4.base-search-card__subtitle",
				Location:     "span.job-search-card__location",
				Description: "div.job-snippet",
				Salary:       "span.salary-info",
				PostedDate:   "time.job-search-card__listdate",
			},
		},
	}

	// Initialize scraper
	scraper := NewJobScraper(platforms)

	// Scrape each platform concurrently
	var wg sync.WaitGroup
	for _, platform := range platforms {
		wg.Add(1)
		go func(p Platform) {
			defer wg.Done()
			scraper.Scrape(p, *jobTitle, *location, p.Filters)
		}(platform)
	}
	wg.Wait()

	// Save results
	if *output == "multipleplatformjobs.json" {
		scraper.SaveToJSON(*output)
	} else {
		scraper.SaveToCSV(*output)
	}

	fmt.Printf("Scraping completed. Total jobs found: %d. Results saved to %s\n", 
		len(scraper.jobs), *output)
}