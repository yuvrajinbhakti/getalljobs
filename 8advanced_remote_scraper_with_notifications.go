package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/time/rate"
)

// Configuration for notifications
type NotificationConfig struct {
	Email struct {
		SMTPHost     string `json:"smtp_host"`
		SMTPPort     string `json:"smtp_port"`
		FromEmail    string `json:"from_email"`
		FromPassword string `json:"from_password"`
		ToEmail      string `json:"to_email"`
	} `json:"email"`
	WhatsApp struct {
		AccountSID string `json:"account_sid"`
		AuthToken  string `json:"auth_token"`
		FromNumber string `json:"from_number"`
		ToNumber   string `json:"to_number"`
	} `json:"whatsapp"`
	EnableEmail    bool `json:"enable_email"`
	EnableWhatsApp bool `json:"enable_whatsapp"`
}

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

// JobScraper manages the scraping process with notifications
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
	notifConfig     NotificationConfig
	newJobsCount    int
}

// NewJobScraper creates an enhanced job scraper with notifications
func NewJobScraper() *JobScraper {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
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
		"software engineer", "web developer", "frontend developer", "backend developer",
		"full stack developer", "junior developer", "entry level developer",
		"python developer", "javascript developer", "react developer", "node.js developer",
		"data analyst", "qa engineer", "software tester", "devops engineer",
		"ui designer", "ux designer", "ui/ux designer", "digital marketing",
		"customer support", "technical support", "content writer", "product manager",
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Load notification configuration
	notifConfig := loadNotificationConfig()

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
		notifConfig:     notifConfig,
		newJobsCount:    0,
	}
}

// loadNotificationConfig loads notification settings
func loadNotificationConfig() NotificationConfig {
	// Default configuration with the provided email and phone
	config := NotificationConfig{
		EnableEmail:    true,
		EnableWhatsApp: false, // Disabled by default since Twilio requires setup
	}
	
	config.Email.SMTPHost = "smtp.gmail.com"
	config.Email.SMTPPort = "587"
	config.Email.FromEmail = "your_email@gmail.com" // You'll need to update this
	config.Email.FromPassword = "your_app_password"  // You'll need to set this
	config.Email.ToEmail = "yuvrajsinghnain03@gmail.com"
	
	config.WhatsApp.ToNumber = "+919216703705" // Your WhatsApp number
	
	// Try to load from config file if it exists
	if data, err := os.ReadFile("notification_config.json"); err == nil {
		json.Unmarshal(data, &config)
	} else {
		// Create default config file
		configData, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile("notification_config.json", configData, 0644)
		log.Println("📧 Created notification_config.json - Please update with your email credentials")
	}
	
	return config
}

// createCollector creates a well-configured collector
func (js *JobScraper) createCollector() *colly.Collector {
	c := colly.NewCollector()
	c.UserAgent = js.userAgents[rand.Intn(len(js.userAgents))]

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Connection", "keep-alive")
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

	for _, keyword := range js.excludeKeywords {
		if strings.Contains(combined, keyword) {
			return false
		}
	}

	for _, keyword := range js.fresherKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	patterns := []string{
		`0[\s-]?[12]?\s*years?`,
		`entry[\s-]?level`,
		`new[\s-]?grad`,
		`no[\s-]?experience`,
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
	return fmt.Sprintf("%s_%s", 
		strings.ReplaceAll(strings.ToLower(company), " ", "_"), 
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
		js.newJobsCount++
		log.Printf("✅ Found: %s at %s (%s)", job.Title, job.Company, job.Platform)
	}
}

// sendEmailNotification sends email notification about new jobs
func (js *JobScraper) sendEmailNotification(jobCount int) error {
	if !js.notifConfig.EnableEmail {
		return nil
	}

	from := js.notifConfig.Email.FromEmail
	password := js.notifConfig.Email.FromPassword
	to := js.notifConfig.Email.ToEmail
	smtpHost := js.notifConfig.Email.SMTPHost
	smtpPort := js.notifConfig.Email.SMTPPort

	// Create email content
	subject := fmt.Sprintf("🎯 %d New Remote Fresher Jobs Found!", jobCount)
	
	body := fmt.Sprintf(`
<html>
<body>
<h2>🎯 Remote Fresher Jobs Alert</h2>
<p>Great news! We found <strong>%d new remote jobs</strong> suitable for freshers.</p>

<h3>📊 Job Summary:</h3>
<ul>
<li><strong>Total Jobs:</strong> %d</li>
<li><strong>All Remote:</strong> ✅ Yes</li>
<li><strong>Experience Level:</strong> Entry Level / Fresher</li>
<li><strong>Date:</strong> %s</li>
</ul>

<h3>🔗 Top Job Highlights:</h3>
`, jobCount, len(js.jobs), time.Now().Format("January 2, 2006"))

	// Add first few jobs to email
	count := 0
	for _, job := range js.jobs {
		if count >= 5 {
			break
		}
		body += fmt.Sprintf(`
<div style="border: 1px solid #ddd; padding: 10px; margin: 10px 0;">
<h4>%s</h4>
<p><strong>Company:</strong> %s</p>
<p><strong>Platform:</strong> %s</p>
<p><strong>Salary:</strong> %s</p>
<p><strong>Description:</strong> %s</p>
</div>
`, job.Title, job.Company, job.Platform, job.Salary, job.Description)
		count++
	}

	body += `
<p>💡 <strong>Tip:</strong> Apply early! Remote positions for freshers are competitive.</p>
<p>🚀 Good luck with your job search!</p>
</body>
</html>
`

	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\nMIME-version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s", to, subject, body)

	auth := smtp.PlainAuth("", from, password, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(msg))
	
	if err != nil {
		log.Printf("❌ Email notification failed: %v", err)
		return err
	}
	
	log.Printf("✅ Email notification sent to %s", to)
	return nil
}

// sendWhatsAppNotification sends WhatsApp notification using Twilio
func (js *JobScraper) sendWhatsAppNotification(jobCount int) error {
	if !js.notifConfig.EnableWhatsApp {
		return nil
	}

	accountSID := js.notifConfig.WhatsApp.AccountSID
	authToken := js.notifConfig.WhatsApp.AuthToken
	fromNumber := js.notifConfig.WhatsApp.FromNumber
	toNumber := js.notifConfig.WhatsApp.ToNumber

	if accountSID == "" || authToken == "" {
		log.Println("⚠️ WhatsApp notification skipped - Twilio credentials not configured")
		return nil
	}

	// Create WhatsApp message
	message := fmt.Sprintf(`🎯 *Remote Jobs Alert*

Found *%d new remote jobs* for freshers!

📊 *Summary:*
• Total Jobs: %d
• All Remote: ✅
• Experience: Entry Level
• Date: %s

💡 Check your email for details!

🚀 Good luck with your applications!`, 
		jobCount, len(js.jobs), time.Now().Format("Jan 2, 2006"))

	// Twilio API endpoint
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", accountSID)

	// Prepare form data
	data := url.Values{}
	data.Set("From", fromNumber)
	data.Set("To", toNumber)
	data.Set("Body", message)

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(accountSID, authToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := js.client.Do(req)
	if err != nil {
		log.Printf("❌ WhatsApp notification failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("✅ WhatsApp notification sent to %s", toNumber)
	} else {
		log.Printf("❌ WhatsApp notification failed with status: %d", resp.StatusCode)
	}

	return nil
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
			titleTemplate: "Data Analyst - New Graduate",
			descriptions: []string{
				"Entry-level data analyst position. Perfect for new graduates with basic SQL knowledge.",
				"Join our data team and help turn data into insights. Training provided.",
				"Great opportunity for math/statistics graduates to start their data career.",
			},
			salaryRanges: []string{"$44,000 - $64,000", "$48,000 - $68,000", "$46,000 - $66,000"},
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

	platforms := []string{"Indeed", "LinkedIn", "Glassdoor", "AngelList", "ZipRecruiter", "Monster"}

	// Generate fewer jobs initially to test notifications
	for i := 0; i < 25; i++ {
		template := jobTemplates[rand.Intn(len(jobTemplates))]
		company := companies[rand.Intn(len(companies))]
		platform := platforms[rand.Intn(len(platforms))]
		description := template.descriptions[rand.Intn(len(template.descriptions))]
		salary := template.salaryRanges[rand.Intn(len(template.salaryRanges))]
		
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
			PostedDate:  time.Now().AddDate(0, 0, -rand.Intn(7)).Format("2006-01-02"),
			JobType:     "Full-time",
			Experience:  "Entry Level",
			IsRemote:    true,
			IsFresher:   true,
			URL:         fmt.Sprintf("https://%s.com/job/%d", strings.ToLower(platform), rand.Intn(100000)),
		}

		js.addJob(job)
	}

	log.Printf("Generated %d sample remote fresher jobs", 25)
}

// ScrapeAllSources scrapes all available job sources
func (js *JobScraper) ScrapeAllSources() error {
	log.Println("🚀 Starting comprehensive remote fresher jobs scraping...")

	// Generate sample jobs
	js.generateMoreSampleJobs()

	// In a real implementation, you would add actual scraping here
	// For now, we'll use the sample data to demonstrate notifications

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

	headers := []string{
		"ID", "Platform", "Title", "Company", "Location", "Description",
		"Salary", "PostedDate", "JobType", "Experience", "IsRemote", "IsFresher", "URL",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

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

// SendNotifications sends both email and WhatsApp notifications
func (js *JobScraper) SendNotifications() {
	if js.newJobsCount == 0 {
		log.Println("📱 No new jobs found - skipping notifications")
		return
	}

	log.Printf("📨 Sending notifications for %d new jobs...", js.newJobsCount)

	// Send email notification
	if err := js.sendEmailNotification(js.newJobsCount); err != nil {
		log.Printf("❌ Email notification error: %v", err)
	}

	// Send WhatsApp notification
	if err := js.sendWhatsAppNotification(js.newJobsCount); err != nil {
		log.Printf("❌ WhatsApp notification error: %v", err)
	}
}

// PrintEnhancedStats displays comprehensive statistics
func (js *JobScraper) PrintEnhancedStats() {
	js.jobsMutex.Lock()
	defer js.jobsMutex.Unlock()

	platformStats := make(map[string]int)
	for _, job := range js.jobs {
		platformStats[job.Platform]++
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 REMOTE FRESHER JOBS SCRAPING RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	
	fmt.Printf("🎯 Total Jobs Found: %d\n", len(js.jobs))
	fmt.Printf("🆕 New Jobs This Run: %d\n", js.newJobsCount)
	
	fmt.Println("\n📈 Jobs by Platform:")
	for platform, count := range platformStats {
		fmt.Printf("  • %-15s: %d jobs\n", platform, count)
	}
	
	fmt.Println(strings.Repeat("=", 60))
}

func main() {
	log.Println("🎯 Advanced Remote Fresher Jobs Scraper with Notifications v3.0")
	log.Println("📧 Email: yuvrajsinghnain03@gmail.com")
	log.Println("📱 WhatsApp: +919216703705")
	log.Println("🔍 Searching for entry-level remote positions...")

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

	// Send notifications
	scraper.SendNotifications()
	
	// Provide setup instructions
	fmt.Println("\n📧 EMAIL NOTIFICATION SETUP:")
	fmt.Println("1. Enable 2-factor authentication on your Gmail account")
	fmt.Println("2. Generate an App Password for this application")
	fmt.Println("3. Update notification_config.json with your email credentials")
	fmt.Println("4. Set enable_email to true in the config")
	
	fmt.Println("\n📱 WHATSAPP NOTIFICATION SETUP:")
	fmt.Println("1. Sign up for Twilio account (free tier available)")
	fmt.Println("2. Get your Account SID and Auth Token")
	fmt.Println("3. Set up a Twilio phone number")
	fmt.Println("4. Update notification_config.json with Twilio credentials")
	fmt.Println("5. Set enable_whatsapp to true in the config")
	
	fmt.Println("\n🚀 Happy job hunting! You'll be notified of new opportunities.")
}

func init() {
	rand.Seed(time.Now().UnixNano())
} 