package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// Job represents a job listing
type Job struct {
	Title       string
	Company     string
	Location    string
	Description string
	URL         string
}

// JobScraper manages web scraping for job listings
type JobScraper struct {
	jobs      []Job
	jobsMutex sync.Mutex
}

// NewJobScraper creates a new JobScraper instance
func NewJobScraper() *JobScraper {
	return &JobScraper{
		jobs: []Job{},
	}
}

// ScrapeIndeed scrapes job listings from Indeed with improved robustness
func (js *JobScraper) ScrapeIndeed(jobTitle, location string) error {
	// Create a new collector with more advanced configurations
	c := colly.NewCollector(
		colly.AllowedDomains("www.indeed.com"),
		colly.MaxDepth(2),
		// More comprehensive user agent
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	// Add more realistic browser headers
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("DNT", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "same-origin")
		r.Headers.Set("Sec-Fetch-User", "?1")
	})

	// Set more comprehensive request handling
	c.SetRequestTimeout(60 * time.Second)
	
	// Add request delay to reduce chance of blocking
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       3 * time.Second,
		RandomDelay: 5 * time.Second,
	})

	// Error handling
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Scraping error: Status %d, URL: %s, Error: %v", 
			r.StatusCode, r.Request.URL, err)
	})

	// Find and extract job listings
	c.OnHTML("div.job_seen_beacon", func(e *colly.HTMLElement) {
		// More robust title extraction
		title := e.ChildText("h2.jobTitle span[title]")
		if title == "" {
			title = e.ChildText("h2.jobTitle")
		}

		// Extract description if possible
		description := e.ChildText("div.job-snippet")

		job := Job{
			Title:       title,
			Company:     e.ChildText("span.companyName"),
			Location:    e.ChildText("div.companyLocation"),
			Description: description,
			URL:         "https://www.indeed.com" + e.ChildAttr("h2.jobTitle a", "href"),
		}

		js.jobsMutex.Lock()
		js.jobs = append(js.jobs, job)
		js.jobsMutex.Unlock()
	})

	// Pagination handling (optional)
	c.OnHTML("a.jobsearch-FooterLinks", func(e *colly.HTMLElement) {
		nextPage := e.Attr("href")
		if nextPage != "" {
			e.Request.Visit(nextPage)
		}
	})

	// Construct the search URL
	searchURL := fmt.Sprintf(
		"https://www.indeed.com/jobs?q=%s&l=%s", 
		strings.ReplaceAll(jobTitle, " ", "+"), 
		strings.ReplaceAll(location, " ", "+"),
	)

	// Visit the page with error handling
	err := c.Visit(searchURL)
	if err != nil {
		return fmt.Errorf("failed to visit Indeed search page: %v", err)
	}

	// Wait for all requests to finish
	c.Wait()

	return nil
}

// SaveToCSV writes scraped jobs to a CSV file
func (js *JobScraper) SaveToCSV(filename string) error {
	// Ensure we have jobs to save
	if len(js.jobs) == 0 {
		return fmt.Errorf("no jobs to save")
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{"Title", "Company", "Location", "Description", "URL"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write job data
	for _, job := range js.jobs {
		record := []string{
			job.Title,
			job.Company,
			job.Location,
			job.Description,
			job.URL,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Create job scraper
	scraper := NewJobScraper()

	// Scrape Indeed
	err := scraper.ScrapeIndeed("Software Engineer", "San Francisco")
	if err != nil {
		log.Printf("Indeed scraping error: %v", err)
		return
	}

	// Save results
	err = scraper.SaveToCSV("job_listings.csv")
	if err != nil {
		log.Printf("Error saving CSV: %v", err)
		return
	}

	fmt.Printf("Scraping completed. Total jobs scraped: %d\n", len(scraper.jobs))
}