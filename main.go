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
	// wd, err := selenium.NewRemote(
	// 	fmt.Sprintf("http://localhost:%d/wd/hub", 4444), 
	// 	caps,
	// )

	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", 4444))

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
	// Extract job details
for _, card := range jobCards {
    titleElement, err := card.FindElement(selenium.ByCSSSelector, ".base-search-card__title")
    if err != nil {
        continue
    }
    title, err := titleElement.Text()
    if err != nil {
        continue
    }

    companyElement, err := card.FindElement(selenium.ByCSSSelector, ".base-search-card__subtitle")
    if err != nil {
        continue
    }
    company, err := companyElement.Text()
    if err != nil {
        continue
    }

    locationElement, err := card.FindElement(selenium.ByCSSSelector, ".job-search-card__location")
    if err != nil {
        continue
    }
    location, err := locationElement.Text()
    if err != nil {
        continue
    }

    job := Job{
        Title:    title,
        Company:  company,
        Location: location,
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
	// proxyList := []string{
	// 	"http://proxy1.example.com:8080",
	// 	"http://proxy2.example.com:8080",
	// 	"http://proxy3.example.com:8080",
	// }
	// proxyRotator := NewProxyRotator(proxyList)

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













// package main

// import (
// 	"encoding/csv"
// 	"fmt"
// 	"log"
// 	"math/rand"
// 	// "net/http"
// 	"os"
// 	"strings"
// 	"sync"
// 	"time"

// 	// "github.com/PuerkitoBio/goquery"
// 	"github.com/gocolly/colly/v2"
// )

// // Job represents a job listing
// type Job struct {
// 	Title       string
// 	Company     string
// 	Location    string
// 	Description string
// 	URL         string
// }

// // JobScraper manages web scraping for job listings
// type JobScraper struct {
// 	jobs      []Job
// 	jobsMutex sync.Mutex
// }

// // NewJobScraper creates a new JobScraper instance
// func NewJobScraper() *JobScraper {
// 	return &JobScraper{
// 		jobs: []Job{},
// 	}
// }

// // randomUserAgent returns a random user agent string
// func randomUserAgent() string {
// 	userAgents := []string{
// 		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
// 		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
// 		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
// 	}
// 	return userAgents[rand.Intn(len(userAgents))]
// }

// // ScrapeIndeed scrapes job listings from Indeed with improved handling
// func (js *JobScraper) ScrapeIndeed(jobTitle, location string, maxPages int) error {
// 	// Create a new collector
// 	c := colly.NewCollector(
// 		colly.AllowedDomains("www.indeed.com"),
// 		colly.MaxDepth(2),
// 	)

// 	// Set request timeout and configurations
// 	c.SetRequestTimeout(30 * time.Second)
// 	c.Async = true

// 	// Randomize user agent
// 	c.OnRequest(func(r *colly.Request) {
// 		r.Headers.Set("User-Agent", randomUserAgent())
// 		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
// 		log.Printf("Visiting %s", r.URL)
// 	})

// 	// Error handling
// 	c.OnError(func(r *colly.Response, err error) {
// 		log.Printf("Request URL: %v failed with response: %v\nError: %v", 
// 			r.Request.URL, r, err)
// 	})

// 	// Find and extract job listings
// 	c.OnHTML(".job_seen_beacon", func(e *colly.HTMLElement) {
// 		job := Job{
// 			Title:    e.ChildText("h2.jobTitle"),
// 			Company:  e.ChildText(".companyName"),
// 			Location: e.ChildText(".companyLocation"),
// 			URL:      e.Request.URL.String(),
// 		}

// 		// Safely add job to slice
// 		js.jobsMutex.Lock()
// 		js.jobs = append(js.jobs, job)
// 		js.jobsMutex.Unlock()
// 	})

// 	// Pagination handling
// 	c.OnHTML("a.page", func(e *colly.HTMLElement) {
// 		nextPage := e.Attr("href")
// 		if nextPage != "" {
// 			e.Request.Visit(e.Request.AbsoluteURL(nextPage))
// 		}
// 	})

// 	// Construct the base search URL
// 	baseURL := "https://www.indeed.com/jobs"
// 	searchQuery := fmt.Sprintf("?q=%s&l=%s", 
// 		strings.ReplaceAll(jobTitle, " ", "+"), 
// 		strings.ReplaceAll(location, " ", "+"))
	
// 	startURLs := []string{baseURL + searchQuery}
	
// 	// Limit pages to prevent excessive scraping
// 	for i := 0; i < maxPages; i++ {
// 		if i > 0 {
// 			startURLs = append(startURLs, fmt.Sprintf("%s&start=%d", baseURL+searchQuery, i*10))
// 		}
// 	}

// 	// Visit starting URLs
// 	for _, url := range startURLs {
// 		err := c.Visit(url)
// 		if err != nil {
// 			log.Printf("Error visiting %s: %v", url, err)
// 		}
		
// 		// Be nice to the server
// 		time.Sleep(2 * time.Second)
// 	}

// 	// Wait for all requests to finish
// 	c.Wait()

// 	return nil
// }

// // SaveToCSV writes scraped jobs to a CSV file
// func (js *JobScraper) SaveToCSV(filename string) error {
// 	file, err := os.Create(filename)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	writer := csv.NewWriter(file)
// 	defer writer.Flush()

// 	// Write header
// 	headers := []string{"Title", "Company", "Location", "URL"}
// 	if err := writer.Write(headers); err != nil {
// 		return err
// 	}

// 	// Write job data
// 	for _, job := range js.jobs {
// 		record := []string{
// 			job.Title,
// 			job.Company,
// 			job.Location,
// 			job.URL,
// 		}
// 		if err := writer.Write(record); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// func main() {
// 	// Seed random number generator
// 	rand.Seed(time.Now().UnixNano())

// 	// Create job scraper
// 	scraper := NewJobScraper()

// 	// Scrape Indeed jobs
// 	jobTitle := "FrontEnd Engineer"
// 	location := "India"
// 	maxPages := 3 // Limit to prevent excessive scraping

// 	fmt.Printf("Scraping %s jobs in %s\n", jobTitle, location)

// 	err := scraper.ScrapeIndeed(jobTitle, location, maxPages)
// 	if err != nil {
// 		log.Fatalf("Scraping failed: %v", err)
// 	}

// 	// Save results
// 	outputFile := "indeed_jobs.csv"
// 	err = scraper.SaveToCSV(outputFile)
// 	if err != nil {
// 		log.Fatalf("Failed to save jobs: %v", err)
// 	}

// 	fmt.Printf("Scraped %d jobs and saved to %s\n", len(scraper.jobs), outputFile)
// }