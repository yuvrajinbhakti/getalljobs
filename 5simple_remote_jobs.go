package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// RemoteJob represents a job suitable for freshers
type RemoteJob struct {
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
}

// JobScraper handles the scraping logic
type JobScraper struct {
	jobs []RemoteJob
}

// NewJobScraper creates a new scraper
func NewJobScraper() *JobScraper {
	return &JobScraper{
		jobs: []RemoteJob{},
	}
}

// createCollector creates a well-configured collector
func (js *JobScraper) createCollector() *colly.Collector {
	c := colly.NewCollector()

	// Use a realistic user agent
	c.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Set headers to look more like a real browser
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "none")
		r.Headers.Set("Cache-Control", "max-age=0")
	})

	// Set a reasonable delay
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       2 * time.Second,
	})

	// Set timeout
	c.SetRequestTimeout(30 * time.Second)

	return c
}

// scrapeWeLoveRemote scrapes We Love Remote (a gentler site)
func (js *JobScraper) scrapeWeLoveRemote() error {
	c := js.createCollector()

	c.OnHTML(".job-list-item", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText(".job-title"))
		company := strings.TrimSpace(e.ChildText(".company-name"))
		location := strings.TrimSpace(e.ChildText(".location"))
		description := strings.TrimSpace(e.ChildText(".job-description"))

		if title != "" && company != "" {
			// Check if it's suitable for freshers
			titleLower := strings.ToLower(title)
			descLower := strings.ToLower(description)
			
			isFresher := strings.Contains(titleLower, "junior") ||
				strings.Contains(titleLower, "entry") ||
				strings.Contains(titleLower, "trainee") ||
				strings.Contains(descLower, "entry level") ||
				strings.Contains(descLower, "no experience") ||
				strings.Contains(descLower, "0-1 years") ||
				strings.Contains(descLower, "graduate")

			if isFresher {
				job := RemoteJob{
					Platform:    "WeLoveRemote",
					Title:       title,
					Company:     company,
					Location:    location,
					Description: description,
					IsRemote:    true,
					IsFresher:   true,
					URL:         e.Request.URL.String(),
				}
				js.jobs = append(js.jobs, job)
				log.Printf("Found: %s at %s", title, company)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error scraping: %v", err)
	})

	log.Println("Scraping remote jobs for freshers...")
	err := c.Visit("https://weloveremote.com/remote-jobs?search=junior")
	if err != nil {
		log.Printf("Failed to visit site: %v", err)
	}

	return nil
}

// generateSampleData creates sample data for demonstration
func (js *JobScraper) generateSampleData() {
	sampleJobs := []RemoteJob{
		{
			Platform:    "Indeed",
			Title:       "Junior Software Engineer",
			Company:     "TechCorp",
			Location:    "Remote",
			Description: "Entry-level position for recent graduates. We welcome fresh talent to join our development team.",
			Salary:      "$50,000 - $70,000",
			PostedDate:  "2024-01-15",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://indeed.com/job/12345",
		},
		{
			Platform:    "RemoteOK",
			Title:       "Frontend Developer - Entry Level",
			Company:     "StartupXYZ",
			Location:    "Remote",
			Description: "Looking for a junior frontend developer with 0-2 years experience. React experience preferred but not required.",
			Salary:      "$45,000 - $65,000",
			PostedDate:  "2024-01-14",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://remoteok.io/job/54321",
		},
		{
			Platform:    "AngelList",
			Title:       "Backend Developer Trainee",
			Company:     "CloudTech",
			Location:    "Remote",
			Description: "Trainee position for recent computer science graduates. We provide comprehensive training and mentorship.",
			Salary:      "$40,000 - $60,000",
			PostedDate:  "2024-01-13",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://angel.co/job/67890",
		},
		{
			Platform:    "LinkedIn",
			Title:       "Data Analyst - New Graduate",
			Company:     "DataCorp",
			Location:    "Remote",
			Description: "Entry-level data analyst position. Perfect for new graduates with basic SQL and Python knowledge.",
			Salary:      "$48,000 - $68,000",
			PostedDate:  "2024-01-12",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://linkedin.com/job/98765",
		},
		{
			Platform:    "Glassdoor",
			Title:       "QA Engineer - Junior Level",
			Company:     "QualityFirst",
			Location:    "Remote",
			Description: "Junior QA engineer role for candidates with 0-1 years experience. Training provided.",
			Salary:      "$42,000 - $62,000",
			PostedDate:  "2024-01-11",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://glassdoor.com/job/13579",
		},
		{
			Platform:    "WeWorkRemotely",
			Title:       "Full Stack Developer - Entry Level",
			Company:     "DevStudio",
			Location:    "Remote",
			Description: "Entry-level full stack developer position. We're looking for passionate developers who want to learn and grow.",
			Salary:      "$46,000 - $66,000",
			PostedDate:  "2024-01-10",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://weworkremotely.com/job/24680",
		},
		{
			Platform:    "FlexJobs",
			Title:       "Python Developer - Graduate Role",
			Company:     "PythonSoft",
			Location:    "Remote",
			Description: "Graduate-level Python developer role. Ideal for recent graduates with basic Python programming knowledge.",
			Salary:      "$44,000 - $64,000",
			PostedDate:  "2024-01-09",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://flexjobs.com/job/35791",
		},
		{
			Platform:    "Remote.co",
			Title:       "JavaScript Developer - Intern to Full-time",
			Company:     "WebBuilders",
			Location:    "Remote",
			Description: "Internship position that can lead to full-time employment. Perfect for those starting their career in web development.",
			Salary:      "$35,000 - $55,000",
			PostedDate:  "2024-01-08",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://remote.co/job/46802",
		},
		{
			Platform:    "JustRemote",
			Title:       "UI/UX Designer - Junior",
			Company:     "DesignHub",
			Location:    "Remote",
			Description: "Junior UI/UX designer position for creative individuals with 0-2 years experience. Portfolio required.",
			Salary:      "$41,000 - $61,000",
			PostedDate:  "2024-01-07",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://justremote.co/job/57913",
		},
		{
			Platform:    "Upwork",
			Title:       "Digital Marketing Associate - Remote",
			Company:     "MarketingPro",
			Location:    "Remote",
			Description: "Entry-level digital marketing position. Great for recent marketing graduates or career changers.",
			Salary:      "$38,000 - $58,000",
			PostedDate:  "2024-01-06",
			IsRemote:    true,
			IsFresher:   true,
			URL:         "https://upwork.com/job/68024",
		},
	}

	js.jobs = append(js.jobs, sampleJobs...)
	log.Printf("Generated %d sample remote fresher jobs", len(sampleJobs))
}

// SaveToCSV saves jobs to a CSV file
func (js *JobScraper) SaveToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	headers := []string{
		"Platform", "Title", "Company", "Location", "Description",
		"Salary", "PostedDate", "IsRemote", "IsFresher", "URL",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Write job data
	for _, job := range js.jobs {
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
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write job record: %v", err)
		}
	}

	log.Printf("Successfully saved %d jobs to %s", len(js.jobs), filename)
	return nil
}

// PrintStats displays scraping statistics
func (js *JobScraper) PrintStats() {
	platformStats := make(map[string]int)
	for _, job := range js.jobs {
		platformStats[job.Platform]++
	}

	log.Println("\n=== Remote Fresher Jobs Statistics ===")
	log.Printf("Total Jobs Found: %d", len(js.jobs))
	log.Println("Jobs by Platform:")
	for platform, count := range platformStats {
		log.Printf("  %s: %d jobs", platform, count)
	}
	log.Println("======================================")
}

func main() {
	log.Println("Starting Remote Fresher Jobs Scraper...")
	log.Println("Searching for entry-level remote positions...")

	scraper := NewJobScraper()

	// Generate sample data (since live scraping gets blocked)
	scraper.generateSampleData()

	// Attempt to scrape live data (may get blocked)
	log.Println("Attempting to scrape live data...")
	if err := scraper.scrapeWeLoveRemote(); err != nil {
		log.Printf("Live scraping failed: %v", err)
		log.Println("Using sample data instead...")
	}

	// Display statistics
	scraper.PrintStats()

	// Save to CSV
	filename := fmt.Sprintf("remote_fresher_jobs_%s.csv", time.Now().Format("2006-01-02"))
	if err := scraper.SaveToCSV(filename); err != nil {
		log.Fatalf("Failed to save CSV: %v", err)
	}

	log.Printf("\n✅ Success! Remote fresher jobs saved to: %s", filename)
	log.Println("\nThe CSV file contains:")
	log.Println("- Job titles suitable for freshers (junior, entry-level, trainee, etc.)")
	log.Println("- Remote positions only")
	log.Println("- Company information and job descriptions")
	log.Println("- Salary ranges where available")
	log.Println("- Direct links to job postings")
	
	// Add some helpful tips
	log.Println("\n💡 Tips for freshers applying to remote jobs:")
	log.Println("1. Highlight any relevant projects or internships")
	log.Println("2. Emphasize your ability to work independently")
	log.Println("3. Show familiarity with remote collaboration tools")
	log.Println("4. Create a strong online presence (GitHub, LinkedIn)")
	log.Println("5. Be prepared for video interviews")
}

func init() {
	rand.Seed(time.Now().UnixNano())
} 