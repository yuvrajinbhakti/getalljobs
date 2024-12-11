package jobscraper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/tebeka/selenium"
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

// ScrapeIndeed scrapes job listings from Indeed
func (js *JobScraper) ScrapeIndeed(jobTitle, location string) error {
	c := colly.NewCollector(
		colly.AllowedDomains("www.indeed.com"),
		colly.MaxDepth(2),
	)

	// Set request timeout and interval
	c.SetRequestTimeout(30 * time.Second)
	c.Async = true

	// Find and extract job listings
	c.OnHTML(".job_seen_beacon", func(e *colly.HTMLElement) {
		job := Job{
			Title:    e.ChildText("h2.jobTitle"),
			Company:  e.ChildText(".companyName"),
			Location: e.ChildText(".companyLocation"),
			URL:      e.Request.URL.String(),
		}

		js.jobsMutex.Lock()
		js.jobs = append(js.jobs, job)
		js.jobsMutex.Unlock()
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %v failed with response: %v\nError: %v", 
			r.Request.URL, r, err)
	})

	// Construct the search URL
	searchURL := fmt.Sprintf(
		"https://www.indeed.com/jobs?q=%s&l=%s", 
		strings.ReplaceAll(jobTitle, " ", "+"), 
		strings.ReplaceAll(location, " ", "+"),
	)

	// Visit the page
	return c.Visit(searchURL)
}

// ScrapeLInkedInSelenium uses Selenium for dynamic content scraping
func (js *JobScraper) ScrapeLInkedInSelenium(jobTitle, location string) error {
	// Configure Selenium WebDriver
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}

	// Connect to the WebDriver
	wd, err := selenium.NewRemoteWebDriver(
		fmt.Sprintf("http://localhost:%d/wd/hub", 4444), 
		caps,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to WebDriver: %v", err)
	}
	defer wd.Quit()

	// Construct search URL
	searchURL := fmt.Sprintf(
		"https://www.linkedin.com/jobs/search/?keywords=%s&location=%s", 
		strings.ReplaceAll(jobTitle, " ", "%20"), 
		strings.ReplaceAll(location, " ", "%20"),
	)

	// Navigate to the page
	if err := wd.Get(searchURL); err != nil {
		return fmt.Errorf("failed to navigate: %v", err)
	}

	// Wait and find job listings
	jobCards, err := wd.FindElements(selenium.ByCSSSelector, ".base-card")
	if err != nil {
		return fmt.Errorf("failed to find job cards: %v", err)
	}

	// Extract job details
	for _, card := range jobCards {
		title, err := card.FindElement(selenium.ByCSSSelector, ".base-search-card__title")
		if err != nil {
			continue
		}

		company, err := card.FindElement(selenium.ByCSSSelector, ".base-search-card__subtitle")
		if err != nil {
			continue
		}

		location, err := card.FindElement(selenium.ByCSSSelector, ".job-search-card__location")
		if err != nil {
			continue
		}

		job := Job{
			Title:    title.Text(),
			Company:  company.Text(),
			Location: location.Text(),
		}

		js.jobsMutex.Lock()
		js.jobs = append(js.jobs, job)
		js.jobsMutex.Unlock()
	}

	return nil
}

// SaveToCSV writes scraped jobs to a CSV file
func (js *JobScraper) SaveToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{"Title", "Company", "Location", "URL"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write job data
	for _, job := range js.jobs {
		record := []string{
			job.Title,
			job.Company,
			job.Location,
			job.URL,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ProxyRotator manages IP rotation for scraping
type ProxyRotator struct {
	proxies []string
	current int
}

// NewProxyRotator creates a new proxy rotator
func NewProxyRotator(proxies []string) *ProxyRotator {
	return &ProxyRotator{
		proxies: proxies,
		current: 0,
	}
}

// GetNextProxy returns the next proxy in the rotation
func (pr *ProxyRotator) GetNextProxy() string {
	proxy := pr.proxies[pr.current]
	pr.current = (pr.current + 1) % len(pr.proxies)
	return proxy
}

// Example main function for demonstration
func main() {
	// Create job scraper
	scraper := NewJobScraper()

	// Proxies for rotation (example list)
	proxyList := []string{
		"http://proxy1.example.com:8080",
		"http://proxy2.example.com:8080",
		"http://proxy3.example.com:8080",
	}
	proxyRotator := NewProxyRotator(proxyList)

	// Example usage
	err := scraper.ScrapeIndeed("Software Engineer", "San Francisco")
	if err != nil {
		log.Fatal(err)
	}

	// Save results
	err = scraper.SaveToCSV("job_listings.csv")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Scraping completed successfully")
}

// Web Scraping Best Practices in Go:
// 1. Use timeouts on HTTP clients
// 2. Implement concurrency safely
// 3. Rotate IP addresses
// 4. Handle errors gracefully
// 5. Respect website terms of service
// 6. Use appropriate parsing libraries