# Remote Fresher Jobs Scraper

This Go application scrapes multiple job platforms to find remote job opportunities specifically for freshers/entry-level candidates.

## Features

- **Multi-Platform Scraping**: Scrapes Indeed, RemoteOK, and other job platforms
- **Fresher-Focused**: Filters jobs based on experience level (entry-level, junior, graduate, etc.)
- **Remote-Only**: Only collects remote job opportunities
- **CSV Export**: Saves results in a structured CSV format
- **Rate Limiting**: Respects website rate limits to avoid blocking
- **Concurrent Scraping**: Uses goroutines for efficient parallel scraping

## Job Platforms Covered

1. **Indeed** - Filters for remote entry-level positions
2. **RemoteOK** - Specialized remote job board
3. **WeWorkRemotely** - Remote-first job board
4. **AngelList** - Startup jobs (remote positions)

## How It Works

1. **Keyword Filtering**: Uses fresher-specific keywords like:
   - "entry level", "junior", "fresher", "graduate", "trainee"
   - "0-1 years", "0-2 years", "recent graduate", "new grad"

2. **Remote Detection**: Identifies remote jobs using keywords:
   - "remote", "work from home", "telecommute", "distributed"
   - "home office", "anywhere", "location independent"

3. **Experience Filtering**: Excludes senior positions with keywords:
   - "senior", "lead", "principal", "architect", "manager"
   - "5+ years", "10+ years", "experienced"

## Job Categories Searched

- Software Engineer
- Web Developer (Frontend/Backend/Full Stack)
- Junior Developer
- Data Analyst
- QA Engineer
- DevOps Engineer
- UI/UX Designer
- Product Manager
- Business Analyst
- Technical Writer

## Output Format

The scraper generates a CSV file with the following columns:
- Platform
- Title
- Company
- Location
- Description
- Salary
- PostedDate
- ExperienceLevel
- Remote (true/false)
- URL
- ApplyURL

## Usage

1. Make sure you have Go installed
2. Install dependencies: `go mod tidy`
3. Run the scraper: `go run remote_fresher_jobs.go`
4. The CSV file will be saved as `remote_fresher_jobs_YYYY-MM-DD.csv`

## Features

- **Rate Limiting**: Respects website policies with appropriate delays
- **Error Handling**: Graceful error handling and logging
- **Concurrent Scraping**: Efficient parallel processing
- **User Agent Rotation**: Mimics real browser requests
- **Duplicate Prevention**: Avoids duplicate job listings

## Configuration

The scraper can be customized by modifying:
- Job search keywords
- Fresher-specific keywords
- Remote job keywords
- Rate limiting parameters
- Maximum pages to scrape per platform

## Legal Note

This scraper is for educational and personal use only. Please respect the terms of service of each job platform and use responsibly. 