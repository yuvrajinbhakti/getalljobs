package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
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

	"github.com/go-redis/redis/v8"
	"github.com/gocolly/colly/v2"
	"golang.org/x/time/rate"
)

// ConfigData holds application configuration
type ConfigData struct {
	WhatsappAPIKey string   `json:"whatsapp_api_key"`
	WhatsappNumber string   `json:"whatsapp_number"`
	RedisURL       string   `json:"redis_url"`
	ProxyList      []string `json:"proxy_list"`
}

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
	Skills       string
}

// Job represents a comprehensive job listing
type Job struct {
	ID          string    // Unique identifier for deduplication
	Platform    string
	Title       string
	Company     string
	Location    string
	Description string
	Salary      string
	PostedDate  string
	URL         string
	Skills      []string
	AddedAt     time.Time
}

// JobScraper manages the scraping process across multiple platforms
type JobScraper struct {
	jobs         []Job
	jobsMutex    sync.Mutex
	rateLimiter  *rate.Limiter
	platforms    []Platform
	collector    *colly.Collector
	redis        *redis.Client
	config       ConfigData
	seenJobs     map[string]bool
	seenJobMutex sync.Mutex
}

// Common user agents for rotation
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
}

// Common technical skills to look for
var commonSkills = []string{
	"python", "java", "javascript", "golang", "react", "aws", "docker",
	"kubernetes", "sql", "nosql", "git", "agile", "rust", "c++",
	"machine learning", "ai", "cloud", "devops", "nodejs", "angular",
}

// Helper Functions
func randomUserAgent() string {
	rand.Seed(time.Now().UnixNano())
	return userAgents[rand.Intn(len(userAgents))]
}

func sanitizeText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	return text
}

func extractSkills(text string) []string {
	text = strings.ToLower(text)
	var foundSkills []string
	
	for _, skill := range commonSkills {
		if strings.Contains(text, strings.ToLower(skill)) {
			foundSkills = append(foundSkills, skill)
		}
	}
	
	return foundSkills
}

// NewJobScraper creates an enhanced JobScraper instance
func NewJobScraper(platforms []Platform, config ConfigData) *JobScraper {
	rdb := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	return &JobScraper{
		jobs:        []Job{},
		rateLimiter: rate.NewLimiter(rate.Every(2*time.Second), 2),
		platforms:   platforms,
		collector:   createCollector(),
		redis:       rdb,
		config:      config,
		seenJobs:    make(map[string]bool),
	}
}

func createCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(3),
		colly.UserAgent(randomUserAgent()),
	)

	c.WithTransport(&http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	})

	return c
}

// JobScraper Methods
func (js *JobScraper) sendWhatsAppMessage(job Job) error {
	message := fmt.Sprintf("🆕 New Job Alert!\n\n"+
		"🏢 Company: %s\n"+
		"💼 Position: %s\n"+
		"📍 Location: %s\n"+
		"💰 Salary: %s\n\n"+
		"🔗 Apply here: %s",
		job.Company, job.Title, job.Location, job.Salary, job.URL)

	payload := map[string]interface{}{
		"phone":   js.config.WhatsappNumber,
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.whatsapp.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+js.config.WhatsappAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("whatsapp API returned status: %d", resp.StatusCode)
	}

	return nil
}

func (js *JobScraper) isNewJob(job Job) bool {
	js.seenJobMutex.Lock()
	defer js.seenJobMutex.Unlock()

	jobID := fmt.Sprintf("%s-%s-%s", job.Platform, job.Company, job.Title)
	if js.seenJobs[jobID] {
		return false
	}

	exists, err := js.redis.Exists(context.Background(), jobID).Result()
	if err != nil {
		log.Printf("Redis error: %v", err)
		return true
	}

	if exists == 1 {
		return false
	}

	js.seenJobs[jobID] = true
	js.redis.Set(context.Background(), jobID, time.Now().String(), 7*24*time.Hour)
	return true
}

func (js *JobScraper) SaveToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Platform", "Title", "Company", "Location", "Salary", 
		"Posted Date", "URL", "Skills", "Added At"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, job := range js.jobs {
		record := []string{
			job.Platform,
			job.Title,
			job.Company,
			job.Location,
			job.Salary,
			job.PostedDate,
			job.URL,
			strings.Join(job.Skills, "|"),
			job.AddedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func (js *JobScraper) Scrape(platform Platform, jobTitle, location string) {
	err := js.rateLimiter.Wait(context.Background())
	if err != nil {
		log.Printf("Rate limit error: %v", err)
		return
	}

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
			Skills:      extractSkills(e.ChildText(platform.Selector.Skills)),
			AddedAt:     time.Now(),
		}

		if job.Title != "" && job.Company != "" && js.isNewJob(job) {
			js.jobsMutex.Lock()
			js.jobs = append(js.jobs, job)
			js.jobsMutex.Unlock()

			if err := js.sendWhatsAppMessage(job); err != nil {
				log.Printf("Failed to send WhatsApp notification: %v", err)
			}
		}
	})

	searchURL := fmt.Sprintf("%s%s?q=%s&l=%s",
		platform.BaseURL,
		platform.QueryPath,
		url.QueryEscape(jobTitle),
		url.QueryEscape(location))

	if err := js.collector.Visit(searchURL); err != nil {
		log.Printf("Failed to visit %s: %v", platform.Name, err)
	}

	js.collector.Wait()
}

// Configuration functions
func loadConfig(path string) (ConfigData, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return ConfigData{}, err
	}

	var config ConfigData
	err = json.Unmarshal(file, &config)
	return config, err
}

func initializePlatforms() []Platform {
	return []Platform{
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
				Skills:       ".skills-section",
			},
		},
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
				Skills:       ".skills-section",
			},
		},
	}
}

func main() {
	jobTitle := flag.String("title", "", "Job title to search for")
	location := flag.String("location", "", "Job location")
	configFile := flag.String("config", "config.json", "Path to configuration file")
	outputFile := flag.String("output", "jobs.csv", "Output CSV file")
	flag.Parse()

	configData, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	platforms := initializePlatforms()
	scraper := NewJobScraper(platforms, configData)
	
	var wg sync.WaitGroup
	for _, platform := range platforms {
		wg.Add(1)
		go func(p Platform) {
			defer wg.Done()
			scraper.Scrape(p, *jobTitle, *location)
		}(platform)
	}
	wg.Wait()

	if err := scraper.SaveToCSV(*outputFile); err != nil {
		log.Fatalf("Failed to save results: %v", err)
	}
}


// go run main.go -title "Software Engineer" -location "Remote" -config config.json