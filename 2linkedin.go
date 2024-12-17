

// package main

// import (
// 	"encoding/csv"
// 	"fmt"
// 	"log"
// 	"math/rand"
// 	"net/http"
// 	"os"
// 	"strings"
// 	"sync"
// 	"time"

// 	"github.com/PuerkitoBio/goquery"
// )

// type Job struct {
// 	Title    string
// 	Company  string
// 	Location string
// 	URL      string
// }

// type JobScraper struct {
// 	jobs      []Job
// 	jobsMutex sync.Mutex
// }

// // ScrapeJobs scrapes jobs from the provided URL.
// func (js *JobScraper) ScrapeJobs(searchURL string) error {
// 	client := &http.Client{}
// 	rand.Seed(time.Now().UnixNano()) // Seed the random number generator for delays

// 	for attempt := 1; attempt <= 5; attempt++ {
// 		req, err := http.NewRequest("GET", searchURL, nil)
// 		if err != nil {
// 			return fmt.Errorf("failed to create request: %w", err)
// 		}
// 		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")

// 		resp, err := client.Do(req)
// 		if err != nil {
// 			return fmt.Errorf("failed to fetch URL: %w", err)
// 		}
// 		defer resp.Body.Close()

// 		if resp.StatusCode == 429 {
// 			log.Printf("Rate limit reached. Retrying in a few seconds... (Attempt %d)", attempt)
// 			time.Sleep(time.Duration(rand.Intn(10)+5) * time.Second)
// 			continue
// 		}

// 		if resp.StatusCode != 200 {
// 			return fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
// 		}

// 		doc, err := goquery.NewDocumentFromReader(resp.Body)
// 		if err != nil {
// 			return fmt.Errorf("failed to parse HTML: %w", err)
// 		}

// 		doc.Find(".base-card").Each(func(i int, s *goquery.Selection) {
// 			title := strings.TrimSpace(s.Find(".base-search-card__title").Text())
// 			company := strings.TrimSpace(s.Find(".base-search-card__subtitle").Text())
// 			location := strings.TrimSpace(s.Find(".job-search-card__location").Text())
// 			url, exists := s.Find("a").Attr("href")
// 			if !exists {
// 				url = searchURL // Fallback to the base URL if no specific link is found
// 			}

// 			if title == "" || company == "" || location == "" {
// 				log.Printf("Skipping incomplete job listing")
// 				return
// 			}

// 			job := Job{
// 				Title:    title,
// 				Company:  company,
// 				Location: location,
// 				URL:      url,
// 			}

// 			js.jobsMutex.Lock()
// 			js.jobs = append(js.jobs, job)
// 			js.jobsMutex.Unlock()
// 		})

// 		time.Sleep(time.Duration(rand.Intn(10)+5) * time.Second) // Random delay
// 		break // Exit the loop if scraping is successful
// 	}

// 	return nil
// }

// // SaveJobsToCSV saves the scraped jobs to a CSV file.
// func (js *JobScraper) SaveJobsToCSV(filename string) error {
// 	file, err := os.Create(filename)
// 	if err != nil {
// 		return fmt.Errorf("failed to create CSV file: %w", err)
// 	}
// 	defer file.Close()

// 	writer := csv.NewWriter(file)
// 	defer writer.Flush()

// 	headers := []string{"Title", "Company", "Location", "URL"}
// 	if err := writer.Write(headers); err != nil {
// 		return fmt.Errorf("failed to write header to CSV: %w", err)
// 	}

// 	for _, job := range js.jobs {
// 		record := []string{job.Title, job.Company, job.Location, job.URL}
// 		if err := writer.Write(record); err != nil {
// 			return fmt.Errorf("failed to write record to CSV: %w", err)
// 		}
// 	}

// 	return nil
// }

// func main() {
// 	searchURL := "https://www.linkedin.com/jobs/search/?currentJobId=4062249320&f_E=2&f_WT=2&origin=JOB_SEARCH_PAGE_JOB_FILTER&refresh=true"

// 	scraper := &JobScraper{}
// 	if err := scraper.ScrapeJobs(searchURL); err != nil {
// 		log.Fatalf("Failed to scrape jobs: %v", err)
// 	}

// 	csvFile := "jobs.csv"
// 	if err := scraper.SaveJobsToCSV(csvFile); err != nil {
// 		log.Fatalf("Failed to save jobs to CSV: %v", err)
// 	}

// 	log.Printf("Jobs successfully saved to %s", csvFile)

// 	// Optional: Print jobs to the console
// 	for _, job := range scraper.jobs {
// 		fmt.Printf("Job Title: %s\nCompany: %s\nLocation: %s\nURL: %s\n\n",
// 			job.Title, job.Company, job.Location, job.URL)
// 	}
// }






package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Job struct {
	Title    string
	Company  string
	Location string
	URL      string
}

type JobScraper struct {
	jobs      []Job
	jobsMutex sync.Mutex
}

// ScrapeJobs scrapes jobs from the provided URL.
func (js *JobScraper) ScrapeJobs(searchURL string) error {
	client := &http.Client{}
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator for delays

	for attempt := 1; attempt <= 5; attempt++ {
		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			log.Printf("Rate limit reached. Retrying in a few seconds... (Attempt %d)", attempt)
			time.Sleep(time.Duration(rand.Intn(10)+5) * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to parse HTML: %w", err)
		}

		doc.Find(".base-card").Each(func(i int, s *goquery.Selection) {
			title := strings.TrimSpace(s.Find(".base-search-card__title").Text())
			company := strings.TrimSpace(s.Find(".base-search-card__subtitle").Text())
			location := strings.TrimSpace(s.Find(".job-search-card__location").Text())
			url, exists := s.Find("a").Attr("href")
			if !exists {
				url = searchURL // Fallback to the base URL if no specific link is found
			}

			if title == "" || company == "" || location == "" {
				log.Printf("Skipping incomplete job listing")
				return
			}

			job := Job{
				Title:    title,
				Company:  company,
				Location: location,
				URL:      url,
			}

			js.jobsMutex.Lock()
			js.jobs = append(js.jobs, job)
			js.jobsMutex.Unlock()
		})

		time.Sleep(time.Duration(rand.Intn(10)+5) * time.Second) // Random delay
		break // Exit the loop if scraping is successful
	}

	return nil
}

// SaveJobsToCSV saves the scraped jobs to a CSV file.
func (js *JobScraper) SaveJobsToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"Title", "Company", "Location", "URL"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write header to CSV: %w", err)
	}

	for _, job := range js.jobs {
		record := []string{job.Title, job.Company, job.Location, job.URL}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record to CSV: %w", err)
		}
	}

	return nil
}

func main() {
	searchURL := "https://www.linkedin.com/jobs/search/?currentJobId=4062249320&f_E=2&f_WT=2&origin=JOB_SEARCH_PAGE_JOB_FILTER&refresh=true"

	scraper := &JobScraper{}
	if err := scraper.ScrapeJobs(searchURL); err != nil {
		log.Fatalf("Failed to scrape jobs: %v", err)
	}

	csvFile := "jobs.csv"
	if err := scraper.SaveJobsToCSV(csvFile); err != nil {
		log.Fatalf("Failed to save jobs to CSV: %v", err)
	}

	log.Printf("Jobs successfully saved to %s", csvFile)

	// Optional: Print jobs to the console
	for _, job := range scraper.jobs {
		fmt.Printf("Job Title: %s\nCompany: %s\nLocation: %s\nURL: %s\n\n",
			job.Title, job.Company, job.Location, job.URL)
	}
}
