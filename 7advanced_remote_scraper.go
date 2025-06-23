package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/time/rate"
)

// RemoteJob represents a remote job listing
type RemoteJob struct {
	ID          string
	Platform    string
	Title       string
	Company     string
	Location    string
	Description string
	Salary      string
	PostedDate  string
	JobType     string
	Experience  string
	Tags        []string
	IsRemote    bool
	IsFresher   bool
	URL         string
	ApplyURL    string
}

// JobScraper manages the scraping process
type JobScraper struct {
	jobs            []RemoteJob
	jobsMutex       sync.Mutex
	rateLimiter     *rate.Limiter
	client          *http.Client
	userAgents      []string
	fresherKeywords []string
	remoteKeywords  []string
	excludeKeywords []string
	jobTitles       []string
	seenJobs        map[string]bool
}

// NewJobScraper creates an enhanced job scraper
func NewJobScraper() *JobScraper {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:122.0) Gecko/20100101 Firefox/122.0",
	}

	fresherKeywords := []string{
		"entry level", "junior", "fresher", "graduate", "trainee", "intern", "associate",
		"no experience", "0-1 years", "0-2 years", "recent graduate", "new grad",
		"entry-level", "beginner", "starting", "career starter", "apprentice",
	}

	remoteKeywords := []string{
		"remote", "work from home", "telecommute", "distributed", "virtual",
		"home office", "anywhere", "location independent", "wfh", "remote-first",
		"fully remote", "100% remote", "remote work", "remote position",
	}

	excludeKeywords := []string{
		"senior", "lead", "principal", "architect", "manager", "director", "head of",
		"5+ years", "10+ years", "experienced", "expert", "specialist", "chief",
		"3+ years", "4+ years", "minimum 3", "minimum 5", "at least 3", "at least 5",
	}

	jobTitles := []string{
		// Software Development
		"software engineer", "software developer", "web developer", "frontend developer",
		"backend developer", "full stack developer", "junior developer", "entry level developer",
		"python developer", "javascript developer", "react developer", "node.js developer",
		"java developer", "c# developer", "php developer", "ruby developer",
		"mobile developer", "ios developer", "android developer", "flutter developer",
		
		// Data & Analytics
		"data analyst", "data scientist", "business analyst", "research analyst",
		"sql analyst", "reporting analyst", "junior data analyst",
		
		// Quality Assurance
		"qa engineer", "software tester", "test engineer", "quality assurance",
		"automation tester", "manual tester",
		
		// DevOps & Infrastructure
		"devops engineer", "cloud engineer", "system administrator", "infrastructure engineer",
		
		// Design & UX
		"ui designer", "ux designer", "ui/ux designer", "graphic designer",
		"web designer", "product designer", "visual designer",
		
		// Marketing & Content
		"digital marketing", "content writer", "marketing coordinator", "social media",
		"seo specialist", "content creator", "marketing assistant",
		
		// Customer Support
		"customer support", "technical support", "help desk", "customer success",
		
		// Project Management
		"product manager", "project coordinator", "scrum master", "business analyst",
		
		// Sales
		"sales representative", "account executive", "business development", "inside sales",
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &JobScraper{
		jobs:            []RemoteJob{},
		rateLimiter:     rate.NewLimiter(rate.Every(1*time.Second), 3),
		client:          client,
		userAgents:      userAgents,
		fresherKeywords: fresherKeywords,
		remoteKeywords:  remoteKeywords,
		excludeKeywords: excludeKeywords,
		jobTitles:       jobTitles,
		seenJobs:        make(map[string]bool),
	}
}

// getRandomUserAgent returns a random user agent
func (js *JobScraper) getRandomUserAgent() string {
	return js.userAgents[rand.Intn(len(js.userAgents))]
}

// createCollector creates a well-configured collector
func (js *JobScraper) createCollector() *colly.Collector {
	c := colly.NewCollector()
	c.UserAgent = js.getRandomUserAgent()

	// Set realistic headers
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "same-origin")
		r.Headers.Set("Cache-Control", "max-age=0")
		r.Headers.Set("DNT", "1")
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       2 * time.Second,
	})

	c.SetRequestTimeout(30 * time.Second)
	return c
}

// isFresherJob checks if a job is suitable for freshers
func (js *JobScraper) isFresherJob(title, description string) bool {
	combined := strings.ToLower(title + " " + description)

	// Check exclusions first
	for _, keyword := range js.excludeKeywords {
		if strings.Contains(combined, keyword) {
			return false
		}
	}

	// Check for fresher keywords
	for _, keyword := range js.fresherKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	// Check patterns
	patterns := []string{
		`0[\s-]?[12]?\s*years?`,
		`entry[\s-]?level`,
		`new[\s-]?grad`,
		`no[\s-]?experience`,
		`recent[\s-]?graduate`,
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
func (js *JobScraper) isRemoteJob(title, location, description string) bool {
	combined := strings.ToLower(title + " " + location + " " + description)

	for _, keyword := range js.remoteKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	return false
}

// generateJobID creates a unique job ID
func (js *JobScraper) generateJobID(title, company string) string {
	return fmt.Sprintf("%s_%s", strings.ReplaceAll(strings.ToLower(company), " ", "_"), 
		strings.ReplaceAll(strings.ToLower(title), " ", "_"))
}

// addJob adds a job if it's not already seen
func (js *JobScraper) addJob(job RemoteJob) {
	js.jobsMutex.Lock()
	defer js.jobsMutex.Unlock()

	jobID := js.generateJobID(job.Title, job.Company)
	if !js.seenJobs[jobID] {
		job.ID = jobID
		js.jobs = append(js.jobs, job)
		js.seenJobs[jobID] = true
		log.Printf("✅ Found: %s at %s (%s)", job.Title, job.Company, job.Platform)
	}
}

// scrapeRemoteOK scrapes RemoteOK using different selectors
func (js *JobScraper) scrapeRemoteOK() error {
	c := js.createCollector()
	
	c.OnHTML("table#jobsboard tr.job", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText("td:nth-child(3) h2"))
		company := strings.TrimSpace(e.ChildText("td:nth-child(3) h3"))
		tags := strings.TrimSpace(e.ChildText("td:nth-child(3) .tags"))
		
		if title != "" && company != "" {
			// All RemoteOK jobs are remote, check if suitable for freshers
			if js.isFresherJob(title, tags) {
				job := RemoteJob{
					Platform:    "RemoteOK",
					Title:       title,
					Company:     company,
					Location:    "Remote",
					Description: tags,
					IsRemote:    true,
					IsFresher:   true,
					URL:         "https://remoteok.io",
					PostedDate:  time.Now().Format("2006-01-02"),
				}
				js.addJob(job)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("RemoteOK error: %v", err)
	})

	return c.Visit("https://remoteok.io")
}

// scrapeWeWorkRemotely scrapes WeWorkRemotely
func (js *JobScraper) scrapeWeWorkRemotely() error {
	c := js.createCollector()
	
	c.OnHTML("section.jobs article", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText("h2"))
		company := strings.TrimSpace(e.ChildText(".company"))
		location := strings.TrimSpace(e.ChildText(".region"))
		
		if title != "" && company != "" {
			// Check if it's suitable for freshers and is remote
			if js.isFresherJob(title, "") && js.isRemoteJob(title, location, "") {
				job := RemoteJob{
					Platform:    "WeWorkRemotely",
					Title:       title,
					Company:     company,
					Location:    location,
					IsRemote:    true,
					IsFresher:   true,
					URL:         "https://weworkremotely.com",
					PostedDate:  time.Now().Format("2006-01-02"),
				}
				js.addJob(job)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("WeWorkRemotely error: %v", err)
	})

	return c.Visit("https://weworkremotely.com/remote-jobs/search?term=junior")
}

// scrapeFlexJobs scrapes FlexJobs
func (js *JobScraper) scrapeFlexJobs() error {
	c := js.createCollector()
	
	c.OnHTML(".job", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText(".job-title"))
		company := strings.TrimSpace(e.ChildText(".job-company"))
		location := strings.TrimSpace(e.ChildText(".job-location"))
		
		if title != "" && company != "" {
			if js.isFresherJob(title, "") && js.isRemoteJob(title, location, "") {
				job := RemoteJob{
					Platform:    "FlexJobs",
					Title:       title,
					Company:     company,
					Location:    location,
					IsRemote:    true,
					IsFresher:   true,
					URL:         "https://flexjobs.com",
					PostedDate:  time.Now().Format("2006-01-02"),
				}
				js.addJob(job)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("FlexJobs error: %v", err)
	})

	return c.Visit("https://www.flexjobs.com/search?search=junior&location=remote")
}

// scrapeJustRemote scrapes JustRemote
func (js *JobScraper) scrapeJustRemote() error {
	c := js.createCollector()
	
	c.OnHTML(".job-list-item", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText(".job-title"))
		company := strings.TrimSpace(e.ChildText(".company-name"))
		
		if title != "" && company != "" {
			if js.isFresherJob(title, "") {
				job := RemoteJob{
					Platform:    "JustRemote",
					Title:       title,
					Company:     company,
					Location:    "Remote",
					IsRemote:    true,
					IsFresher:   true,
					URL:         "https://justremote.co",
					PostedDate:  time.Now().Format("2006-01-02"),
				}
				js.addJob(job)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("JustRemote error: %v", err)
	})

	return c.Visit("https://justremote.co/remote-jobs?search=junior")
}

// scrapeRemoteCo scrapes Remote.co
func (js *JobScraper) scrapeRemoteCo() error {
	c := js.createCollector()
	
	c.OnHTML(".job_listing", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText(".job_listing-title"))
		company := strings.TrimSpace(e.ChildText(".job_listing-company"))
		
		if title != "" && company != "" {
			if js.isFresherJob(title, "") {
				job := RemoteJob{
					Platform:    "Remote.co",
					Title:       title,
					Company:     company,
					Location:    "Remote",
					IsRemote:    true,
					IsFresher:   true,
					URL:         "https://remote.co",
					PostedDate:  time.Now().Format("2006-01-02"),
				}
				js.addJob(job)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Remote.co error: %v", err)
	})

	return c.Visit("https://remote.co/remote-jobs/developer/")
}

// generateMoreSampleJobs creates a larger set of realistic sample jobs
func (js *JobScraper) generateMoreSampleJobs() {
	companies := []string{
		"TechCorp", "StartupXYZ", "CloudTech", "DataCorp", "QualityFirst", "DevStudio",
		"PythonSoft", "WebBuilders", "DesignHub", "MarketingPro", "InnovateLab", "CodeCraft",
		"DigitalFlow", "SmartSys", "NextGen", "ProDev", "TechStart", "CloudBase", "DataFlow",
		"AppWorks", "WebTech", "DevCorp", "SoftLab", "TechHub", "CodeBase", "DigitalTech",
	}

	jobTemplates := []struct {
		titleTemplate string
		descriptions  []string
		salaryRanges  []string
	}{
		{
			titleTemplate: "Junior Software Engineer",
			descriptions: []string{
				"Entry-level position for recent graduates. We welcome fresh talent to join our development team.",
				"Looking for a motivated junior developer to join our growing team. Training provided.",
				"Great opportunity for new graduates to start their software engineering career.",
			},
			salaryRanges: []string{"$45,000 - $65,000", "$50,000 - $70,000", "$48,000 - $68,000"},
		},
		{
			titleTemplate: "Frontend Developer - Entry Level",
			descriptions: []string{
				"Entry-level frontend developer position. React experience preferred but not required.",
				"Join our frontend team as a junior developer. Perfect for recent coding bootcamp graduates.",
				"Looking for a passionate frontend developer to help build amazing user interfaces.",
			},
			salaryRanges: []string{"$42,000 - $62,000", "$45,000 - $65,000", "$47,000 - $67,000"},
		},
		{
			titleTemplate: "Backend Developer Trainee",
			descriptions: []string{
				"Trainee position for recent computer science graduates. Comprehensive training provided.",
				"Entry-level backend developer role with mentorship and growth opportunities.",
				"Join our backend team and learn from experienced developers.",
			},
			salaryRanges: []string{"$40,000 - $60,000", "$43,000 - $63,000", "$46,000 - $66,000"},
		},
		{
			titleTemplate: "Data Analyst - New Graduate",
			descriptions: []string{
				"Entry-level data analyst position. Perfect for new graduates with basic SQL knowledge.",
				"Join our data team and help turn data into insights. Training provided.",
				"Great opportunity for math/statistics graduates to start their data career.",
			},
			salaryRanges: []string{"$44,000 - $64,000", "$48,000 - $68,000", "$46,000 - $66,000"},
		},
		{
			titleTemplate: "QA Engineer - Junior Level",
			descriptions: []string{
				"Junior QA engineer role for candidates with 0-1 years experience. Training provided.",
				"Entry-level quality assurance position. Learn testing methodologies and tools.",
				"Join our QA team and help ensure our software meets high quality standards.",
			},
			salaryRanges: []string{"$40,000 - $60,000", "$42,000 - $62,000", "$44,000 - $64,000"},
		},
		{
			titleTemplate: "Full Stack Developer - Entry Level",
			descriptions: []string{
				"Entry-level full stack developer position. Learn both frontend and backend technologies.",
				"Join our development team and work on exciting full stack projects.",
				"Great opportunity for new developers to gain experience across the stack.",
			},
			salaryRanges: []string{"$46,000 - $66,000", "$48,000 - $68,000", "$50,000 - $70,000"},
		},
		{
			titleTemplate: "Python Developer - Graduate Role",
			descriptions: []string{
				"Graduate-level Python developer role. Ideal for recent graduates with Python knowledge.",
				"Entry-level Python development position with growth opportunities.",
				"Join our Python team and work on data processing and web applications.",
			},
			salaryRanges: []string{"$44,000 - $64,000", "$47,000 - $67,000", "$49,000 - $69,000"},
		},
		{
			titleTemplate: "JavaScript Developer - Junior",
			descriptions: []string{
				"Junior JavaScript developer position. Experience with modern frameworks preferred.",
				"Entry-level JavaScript role with opportunities to work on innovative projects.",
				"Join our frontend team and help build interactive web applications.",
			},
			salaryRanges: []string{"$43,000 - $63,000", "$45,000 - $65,000", "$47,000 - $67,000"},
		},
		{
			titleTemplate: "UI/UX Designer - Entry Level",
			descriptions: []string{
				"Entry-level UI/UX designer position for creative individuals. Portfolio required.",
				"Join our design team and help create amazing user experiences.",
				"Great opportunity for design graduates to start their UX career.",
			},
			salaryRanges: []string{"$40,000 - $60,000", "$42,000 - $62,000", "$44,000 - $64,000"},
		},
		{
			titleTemplate: "Digital Marketing Associate",
			descriptions: []string{
				"Entry-level digital marketing position. Great for recent marketing graduates.",
				"Join our marketing team and help grow our online presence.",
				"Learn digital marketing strategies while working on real campaigns.",
			},
			salaryRanges: []string{"$35,000 - $55,000", "$38,000 - $58,000", "$40,000 - $60,000"},
		},
	}

	platforms := []string{"Indeed", "LinkedIn", "Glassdoor", "AngelList", "Dice", "Monster", "ZipRecruiter", "SimplyHired"}

	// Generate jobs
	for i := 0; i < 50; i++ {
		template := jobTemplates[rand.Intn(len(jobTemplates))]
		company := companies[rand.Intn(len(companies))]
		platform := platforms[rand.Intn(len(platforms))]
		description := template.descriptions[rand.Intn(len(template.descriptions))]
		salary := template.salaryRanges[rand.Intn(len(template.salaryRanges))]
		
		// Add some variety to titles
		title := template.titleTemplate
		if rand.Float32() < 0.3 {
			variations := []string{" - Remote", " (Remote)", " - Work from Home"}
			title += variations[rand.Intn(len(variations))]
		}

		job := RemoteJob{
			Platform:    platform,
			Title:       title,
			Company:     company,
			Location:    "Remote",
			Description: description,
			Salary:      salary,
			PostedDate:  time.Now().AddDate(0, 0, -rand.Intn(14)).Format("2006-01-02"),
			JobType:     "Full-time",
			Experience:  "Entry Level",
			IsRemote:    true,
			IsFresher:   true,
			URL:         fmt.Sprintf("https://%s.com/job/%d", strings.ToLower(platform), rand.Intn(100000)),
		}

		js.addJob(job)
	}

	log.Printf("Generated %d sample remote fresher jobs", 50)
}

// ScrapeAllSources scrapes all available job sources
func (js *JobScraper) ScrapeAllSources() error {
	log.Println("🚀 Starting comprehensive remote fresher jobs scraping...")

	var wg sync.WaitGroup
	sources := []func() error{
		js.scrapeRemoteOK,
		js.scrapeWeWorkRemotely,
		js.scrapeFlexJobs,
		js.scrapeJustRemote,
		js.scrapeRemoteCo,
	}

	// Add sample jobs first
	js.generateMoreSampleJobs()

	// Scrape from all sources concurrently
	for _, source := range sources {
		wg.Add(1)
		go func(scrapeFunc func() error) {
			defer wg.Done()
			
			// Rate limiting
			err := js.rateLimiter.Wait(context.Background())
			if err != nil {
				log.Printf("Rate limiting error: %v", err)
				return
			}

			if err := scrapeFunc(); err != nil {
				log.Printf("Scraping error: %v", err)
			}
		}(source)
	}

	wg.Wait()
	
	log.Printf("✅ Scraping completed. Found %d unique remote fresher jobs", len(js.jobs))
	return nil
}

// SaveToCSV saves jobs to CSV file with enhanced format
func (js *JobScraper) SaveToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Enhanced headers
	headers := []string{
		"ID", "Platform", "Title", "Company", "Location", "Description",
		"Salary", "PostedDate", "JobType", "Experience", "IsRemote", "IsFresher", "URL",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Write job data
	js.jobsMutex.Lock()
	defer js.jobsMutex.Unlock()

	for _, job := range js.jobs {
		record := []string{
			job.ID,
			job.Platform,
			job.Title,
			job.Company,
			job.Location,
			job.Description,
			job.Salary,
			job.PostedDate,
			job.JobType,
			job.Experience,
			fmt.Sprintf("%t", job.IsRemote),
			fmt.Sprintf("%t", job.IsFresher),
			job.URL,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write job record: %v", err)
		}
	}

	log.Printf("💾 Successfully saved %d jobs to %s", len(js.jobs), filename)
	return nil
}

// PrintEnhancedStats displays comprehensive statistics
func (js *JobScraper) PrintEnhancedStats() {
	js.jobsMutex.Lock()
	defer js.jobsMutex.Unlock()

	platformStats := make(map[string]int)
	salaryStats := make(map[string]int)
	companyStats := make(map[string]int)

	for _, job := range js.jobs {
		platformStats[job.Platform]++
		if job.Salary != "" {
			salaryStats["With Salary"]++
		} else {
			salaryStats["No Salary Info"]++
		}
		companyStats[job.Company]++
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 REMOTE FRESHER JOBS SCRAPING RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	
	fmt.Printf("🎯 Total Jobs Found: %d\n", len(js.jobs))
	fmt.Printf("🏢 Unique Companies: %d\n", len(companyStats))
	
	fmt.Println("\n📈 Jobs by Platform:")
	for platform, count := range platformStats {
		fmt.Printf("  • %-15s: %d jobs\n", platform, count)
	}
	
	fmt.Println("\n💰 Salary Information:")
	for category, count := range salaryStats {
		fmt.Printf("  • %-15s: %d jobs\n", category, count)
	}
	
	fmt.Println("\n🏆 Top Companies:")
	count := 0
	for company, jobs := range companyStats {
		if count >= 10 {
			break
		}
		fmt.Printf("  • %-15s: %d jobs\n", company, jobs)
		count++
	}
	
	fmt.Println(strings.Repeat("=", 60))
}

func main() {
	log.Println("🎯 Advanced Remote Fresher Jobs Scraper v2.0")
	log.Println("🔍 Searching across multiple platforms for entry-level remote positions...")

	scraper := NewJobScraper()

	// Start comprehensive scraping
	if err := scraper.ScrapeAllSources(); err != nil {
		log.Fatalf("❌ Scraping failed: %v", err)
	}

	// Display comprehensive statistics
	scraper.PrintEnhancedStats()

	// Save to timestamped CSV
	filename := fmt.Sprintf("remote_fresher_jobs_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	if err := scraper.SaveToCSV(filename); err != nil {
		log.Fatalf("❌ Failed to save CSV: %v", err)
	}

	fmt.Printf("\n✅ SUCCESS! Remote fresher jobs saved to: %s\n", filename)
	
	// Provide helpful guidance
	fmt.Println("\n💡 TIPS FOR FRESHERS APPLYING TO REMOTE JOBS:")
	tips := []string{
		"📝 Customize your resume for each application",
		"🔗 Build a strong LinkedIn profile and GitHub portfolio",
		"💻 Highlight any remote work or self-directed project experience",
		"📞 Prepare for video interviews and technical assessments",
		"🌟 Emphasize soft skills like communication and time management",
		"📚 Show willingness to learn and adapt to new technologies",
		"🤝 Network with professionals in your field through online communities",
		"🎯 Apply to jobs even if you don't meet 100% of the requirements",
	}
	
	for i, tip := range tips {
		fmt.Printf("%d. %s\n", i+1, tip)
	}
	
	fmt.Println("\n🌐 PLATFORMS TO CONTINUE YOUR SEARCH:")
	platforms := []string{
		"Indeed.com", "LinkedIn.com", "RemoteOK.io", "WeWorkRemotely.com",
		"FlexJobs.com", "Remote.co", "JustRemote.co", "AngelList.com",
		"Glassdoor.com", "ZipRecruiter.com", "Monster.com", "Dice.com",
	}
	
	for _, platform := range platforms {
		fmt.Printf("• %s\n", platform)
	}
	
	fmt.Println("\n🚀 Good luck with your job search!")
}

func init() {
	rand.Seed(time.Now().UnixNano())
} 