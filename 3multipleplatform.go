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
	"golang.org/x/time/rate"
)

// Platform defines the configuration for scraping a specific job platform
type Platform struct {
	Name      string
	BaseURL   string
	QueryPath string
	Filters   map[string]string
	Selector  PlatformSelector
}

// PlatformSelector defines CSS selectors for extracting job information
type PlatformSelector struct {
	JobContainer string
	Title        string
	Company      string
	Location     string
	Description  string
	Salary       string
	PostedDate   string
}

// Job represents a single job listing
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

// JobScraper manages the scraping process across multiple platforms
type JobScraper struct {
	jobs        []Job
	jobsMutex   sync.Mutex
	rateLimiter *rate.Limiter
	platforms   []Platform
	collector   *colly.Collector
}

// NewJobScraper creates a new JobScraper with configured rate limiting
func NewJobScraper(platforms []Platform) *JobScraper {
	rateLimiter := rate.NewLimiter(rate.Every(2*time.Second), 2)
	collector := createCollector()
	return &JobScraper{
		jobs:        []Job{},
		rateLimiter: rateLimiter,
		platforms:   platforms,
		collector:   collector,
	}
}

// randomUserAgent returns a random user agent to mimic browser requests
func randomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

// createCollector sets up a Colly collector with advanced configurations
func createCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(3),
		colly.UserAgent(randomUserAgent()),
		colly.AllowURLRevisit(),
	)

	// Set up proxy rotation (optional)
	// c.SetProxy("http://proxy-ip:port")

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

// Scrape performs web scraping for a specific platform
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

// sanitizeText removes unnecessary whitespace
func sanitizeText(text string) string {
	return strings.TrimSpace(text)
}

// SaveToCSV exports job listings to a CSV file
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

func main() {
	// Command-line flags
	jobTitle := flag.String("title", "software engineer", "Job title to search for")
	location := flag.String("location", "remote", "Job location to search for")
	outputFile := flag.String("output", "multiplatformjobs.csv", "Output file for job results")
	flag.Parse()

	// Define multiple job platforms
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
		{
			Name:      "LinkedIn",
			BaseURL:   "https://www.linkedin.com",
			QueryPath: "/jobs/search",
			Selector: PlatformSelector{
				JobContainer: ".base-card",
				Title:        ".base-search-card__title",
				Company:      ".base-search-card__subtitle",
				Location:     ".job-search-card__location",
				Description:  ".job-description",
				Salary:       ".salary-info",
				PostedDate:   ".listed-time",
			},
		},
		{
			Name:      "Glassdoor",
			BaseURL:   "https://www.glassdoor.com",
			QueryPath: "/Job/jobs.htm",
			Selector: PlatformSelector{
				JobContainer: ".react-job-listing",
				Title:        ".job-title",
				Company:      ".job-employer",
				Location:     ".job-location",
				Description:  ".job-description",
				Salary:       ".salary-info",
				PostedDate:   ".job-posted",
			},
		},
	}

	// Create and run scraper
	scraper := NewJobScraper(platforms)

	var wg sync.WaitGroup
	for _, platform := range platforms {
		wg.Add(1)
		go func(p Platform) {
			defer wg.Done()
			scraper.Scrape(p, *jobTitle, *location, p.Filters)
		}(platform)
	}
	wg.Wait()

	// Save results to CSV
	scraper.SaveToCSV(*outputFile)
}