package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/time/rate"
)

// FresherJob represents a remote job suitable for freshers
type FresherJob struct {
	Platform    string
	Title       string
	Company     string
	Location    string
	Description string
	Salary      string
	PostedDate  string
	IsRemote    bool
	IsFresher   bool
	URL         string
	ApplyURL    string
}

// FresherJobScraper manages the scraping process
type FresherJobScraper struct {
	jobs            []FresherJob
	jobsMutex       sync.Mutex
	rateLimiter     *rate.Limiter
	fresherKeywords []string
	remoteKeywords  []string
	excludeKeywords []string
}

// NewFresherJobScraper creates a new scraper instance
func NewFresherJobScraper() *FresherJobScraper {
	fresherKeywords := []string{
		"entry level", "junior", "fresher", "graduate", "trainee", "intern",
		"no experience", "0-1 years", "0-2 years", "recent graduate",
		"entry-level", "beginner", "associate", "new grad", "starting",
	}

	remoteKeywords := []string{
		"remote", "work from home", "telecommute", "distributed",
		"home office", "anywhere", "location independent", "wfh",
	}

	excludeKeywords := []string{
		"senior", "lead", "principal", "architect", "manager", "director",
		"5+ years", "10+ years", "experienced", "expert", "specialist",
	}

	return &FresherJobScraper{
		jobs:            []FresherJob{},
		rateLimiter:     rate.NewLimiter(rate.Every(3*time.Second), 2),
		fresherKeywords: fresherKeywords,
		remoteKeywords:  remoteKeywords,
		excludeKeywords: excludeKeywords,
	}
}

// getRandomUserAgent returns a random user agent
func getRandomUserAgent() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
	}
	return agents[rand.Intn(len(agents))]
}

// createCollector creates a configured colly collector
func (fjs *FresherJobScraper) createCollector() *colly.Collector {
	c := colly.NewCollector(colly.Async(true))
	c.UserAgent = getRandomUserAgent()

	c.WithTransport(&http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       3 * time.Second,
	})

	return c
}

// isFresherJob checks if a job is suitable for freshers
func (fjs *FresherJobScraper) isFresherJob(title, description string) bool {
	combined := strings.ToLower(title + " " + description)

	// Check exclusions first
	for _, keyword := range fjs.excludeKeywords {
		if strings.Contains(combined, keyword) {
			return false
		}
	}

	// Check for fresher keywords
	for _, keyword := range fjs.fresherKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	// Check patterns
	patterns := []string{
		`0[\s-]?[12]?\s*years?`,
		`entry[\s-]?level`,
		`new[\s-]?grad`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, combined)
		if matched {
			return true
		}
	}

	return false
}

// isRemoteJob checks if a job is remote
func (fjs *FresherJobScraper) isRemoteJob(title, location, description string) bool {
	combined := strings.ToLower(title + " " + location + " " + description)

	for _, keyword := range fjs.remoteKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	return false
}

// scrapeIndeed scrapes Indeed for remote fresher jobs
func (fjs *FresherJobScraper) scrapeIndeed(jobTitles []string) error {
	c := fjs.createCollector()

	c.OnHTML("[data-jk]", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText("h2 a span"))
		if title == "" {
			title = strings.TrimSpace(e.ChildText("h2 span"))
		}
		
		company := strings.TrimSpace(e.ChildText("span.companyName"))
		location := strings.TrimSpace(e.ChildText("div.companyLocation"))
		description := strings.TrimSpace(e.ChildText("div.job-snippet"))
		salary := strings.TrimSpace(e.ChildText("span.salaryText"))
		postedDate := strings.TrimSpace(e.ChildText("span.date"))

		applyLink := e.ChildAttr("h2 a", "href")
		if applyLink != "" && !strings.HasPrefix(applyLink, "http") {
			applyLink = "https://www.indeed.com" + applyLink
		}

		if title != "" && company != "" {
			isFresher := fjs.isFresherJob(title, description)
			isRemote := fjs.isRemoteJob(title, location, description)

			if isFresher && isRemote {
				job := FresherJob{
					Platform:    "Indeed",
					Title:       title,
					Company:     company,
					Location:    location,
					Description: description,
					Salary:      salary,
					PostedDate:  postedDate,
					IsRemote:    isRemote,
					IsFresher:   isFresher,
					URL:         e.Request.URL.String(),
					ApplyURL:    applyLink,
				}

				fjs.jobsMutex.Lock()
				fjs.jobs = append(fjs.jobs, job)
				fjs.jobsMutex.Unlock()

				log.Printf("Found: %s at %s", title, company)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error: %v", err)
	})

	for _, jobTitle := range jobTitles {
		searchURL := fmt.Sprintf("https://www.indeed.com/jobs?q=%s&l=Remote&explvl=entry_level&fromage=7",
			url.QueryEscape(jobTitle))

		log.Printf("Scraping Indeed for: %s", jobTitle)
		
		err := fjs.rateLimiter.Wait(context.Background())
		if err != nil {
			return err
		}

		err = c.Visit(searchURL)
		if err != nil {
			log.Printf("Error visiting %s: %v", searchURL, err)
		}
	}

	c.Wait()
	return nil
}

// scrapeRemoteOK scrapes RemoteOK for fresher jobs
func (fjs *FresherJobScraper) scrapeRemoteOK(jobTitles []string) error {
	c := fjs.createCollector()

	c.OnHTML("tr.job", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText("td.company h2"))
		company := strings.TrimSpace(e.ChildText("td.company h3"))
		description := strings.TrimSpace(e.ChildText("td.company .description"))
		salary := strings.TrimSpace(e.ChildText("td.salary"))
		
		applyLink := e.ChildAttr("td.company a", "href")
		if applyLink != "" && !strings.HasPrefix(applyLink, "http") {
			applyLink = "https://remoteok.io" + applyLink
		}

		if title != "" && company != "" {
			isFresher := fjs.isFresherJob(title, description)

			if isFresher {
				job := FresherJob{
					Platform:    "RemoteOK",
					Title:       title,
					Company:     company,
					Location:    "Remote",
					Description: description,
					Salary:      salary,
					PostedDate:  "",
					IsRemote:    true,
					IsFresher:   isFresher,
					URL:         e.Request.URL.String(),
					ApplyURL:    applyLink,
				}

				fjs.jobsMutex.Lock()
				fjs.jobs = append(fjs.jobs, job)
				fjs.jobsMutex.Unlock()

				log.Printf("Found on RemoteOK: %s at %s", title, company)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("RemoteOK Error: %v", err)
	})

	for _, jobTitle := range jobTitles {
		searchURL := fmt.Sprintf("https://remoteok.io/remote-dev-jobs?search=%s",
			url.QueryEscape(jobTitle))

		log.Printf("Scraping RemoteOK for: %s", jobTitle)
		
		err := fjs.rateLimiter.Wait(context.Background())
		if err != nil {
			return err
		}

		err = c.Visit(searchURL)
		if err != nil {
			log.Printf("Error visiting RemoteOK %s: %v", searchURL, err)
		}
	}

	c.Wait()
	return nil
}

// ScrapeAll scrapes all platforms for remote fresher jobs
func (fjs *FresherJobScraper) ScrapeAll(jobTitles []string) error {
	log.Println("Starting Remote Fresher Jobs Scraper...")

	var wg sync.WaitGroup

	// Scrape Indeed
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fjs.scrapeIndeed(jobTitles); err != nil {
			log.Printf("Indeed scraping error: %v", err)
		}
	}()

	// Scrape RemoteOK
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fjs.scrapeRemoteOK(jobTitles); err != nil {
			log.Printf("RemoteOK scraping error: %v", err)
		}
	}()

	wg.Wait()
	
	log.Printf("Scraping completed. Found %d remote fresher jobs", len(fjs.jobs))
	return nil
}

// SaveToCSV saves jobs to CSV file
func (fjs *FresherJobScraper) SaveToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Headers
	headers := []string{
		"Platform", "Title", "Company", "Location", "Description",
		"Salary", "PostedDate", "IsRemote", "IsFresher", "URL", "ApplyURL",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Data
	fjs.jobsMutex.Lock()
	defer fjs.jobsMutex.Unlock()

	for _, job := range fjs.jobs {
		record := []string{
			job.Platform,
			job.Title,
			job.Company,
			job.Location,
			job.Description,
			job.Salary,
			job.PostedDate,
			fmt.Sprintf("%t", job.IsRemote),
			fmt.Sprintf("%t", job.IsFresher),
			job.URL,
			job.ApplyURL,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	log.Printf("Saved %d jobs to %s", len(fjs.jobs), filename)
	return nil
}

// GetStats returns scraping statistics
func (fjs *FresherJobScraper) GetStats() {
	fjs.jobsMutex.Lock()
	defer fjs.jobsMutex.Unlock()

	stats := make(map[string]int)
	for _, job := range fjs.jobs {
		stats[job.Platform]++
	}

	log.Println("=== Scraping Statistics ===")
	log.Printf("Total Jobs: %d", len(fjs.jobs))
	for platform, count := range stats {
		log.Printf("%s: %d jobs", platform, count)
	}
	log.Println("===========================")
}

func runScraper() {
	scraper := NewFresherJobScraper()

	// Job titles to search for
	jobTitles := []string{
		"software engineer",
		"web developer",
		"frontend developer",
		"backend developer",
		"full stack developer",
		"junior developer",
		"entry level developer",
		"python developer",
		"javascript developer",
		"react developer",
		"data analyst",
		"qa engineer",
		"devops engineer",
		"ui ux designer",
		"product manager",
		"business analyst",
	}

	// Start scraping
	if err := scraper.ScrapeAll(jobTitles); err != nil {
		log.Fatalf("Scraping failed: %v", err)
	}

	// Show stats
	scraper.GetStats()

	// Save to CSV
	filename := fmt.Sprintf("remote_fresher_jobs_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	if err := scraper.SaveToCSV(filename); err != nil {
		log.Fatalf("Failed to save CSV: %v", err)
	}

	log.Printf("Success! Results saved to %s", filename)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	runScraper()
} 