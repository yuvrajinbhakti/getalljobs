package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
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
	collector   *colly.Collector
	jobs        []Job
	jobsMutex   sync.Mutex
	rateLimiter *rate.Limiter
	platforms   []Platform
}

// NewJobScraper initializes the scraper with advanced configurations
func NewJobScraper(platforms []Platform) *JobScraper {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(3),
		colly.UserAgent(randomUserAgent()),
	)

	c.SetRequestTimeout(60 * time.Second)

	rateLimiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 10) // 10 requests/sec

	return &JobScraper{
		collector:   c,
		jobs:        []Job{},
		rateLimiter: rateLimiter,
		platforms:   platforms,
	}
}

// randomUserAgent generates random user agents for requests
func randomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

// Scrape scrapes jobs from a specific platform
func (js *JobScraper) Scrape(platform Platform, jobTitle, location string, filters map[string]string) {
	js.collector.OnRequest(func(r *colly.Request) {
		js.rateLimiter.Wait(context.Background())
		log.Printf("Visiting %s", r.URL)
	})

	js.collector.OnError(func(r *colly.Response, err error) {
		log.Printf("Error: %s, Status Code: %d", err.Error(), r.StatusCode)
	})

	js.collector.OnHTML(".job_seen_beacon", func(e *colly.HTMLElement) {
		job := Job{
			Platform:    platform.Name,
			Title:       e.ChildText("h2.jobTitle"),
			Company:     e.ChildText(".companyName"),
			Location:    e.ChildText(".companyLocation"),
			Description: e.ChildText(".job-snippet"),
			Salary:      e.ChildText(".salary-snippet-container"),
			PostedDate:  e.ChildText(".metadata.turnstileId .date"),
			URL:         e.Request.URL.String(),
		}
		js.jobsMutex.Lock()
		js.jobs = append(js.jobs, job)
		js.jobsMutex.Unlock()
	})

	baseURL := fmt.Sprintf("%s%s?q=%s&l=%s", platform.BaseURL, platform.QueryPath, url.QueryEscape(jobTitle), url.QueryEscape(location))

	for key, value := range filters {
		baseURL += fmt.Sprintf("&%s=%s", key, url.QueryEscape(value))
	}

	err := js.collector.Visit(baseURL)
	if err != nil {
		log.Printf("Failed to visit %s: %v", baseURL, err)
	}

	js.collector.Wait()
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
	// Command-line inputs
	jobTitle := flag.String("jobTitle", "Software Engineer", "Job title to search")
	location := flag.String("location", "Remote", "Location to search jobs")
	output := flag.String("output", "multipleplatformjobs.csv", "Output file name (CSV/JSON)")
	flag.Parse()

	// Platforms to scrape
	platforms := []Platform{
		{"Indeed", "https://www.indeed.com", "/jobs", map[string]string{}},
		{"LinkedIn", "https://www.linkedin.com", "/jobs/search", map[string]string{"f_TPR": "r2592000"}}, // e.g., last 30 days filter
	}

	// Initialize scraper
	scraper := NewJobScraper(platforms)

	// Scrape each platform
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

	fmt.Printf("Scraping completed. Results saved to %s\n", *output)
}
