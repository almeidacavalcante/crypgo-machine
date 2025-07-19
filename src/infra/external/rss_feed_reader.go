package external

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RSSFeedReader struct {
	httpClient *http.Client
}

// RSS Feed structure based on standard RSS 2.0
type RSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// Processed news item for sentiment analysis
type NewsItem struct {
	Title       string
	Description string
	Link        string
	Source      string
	PublishedAt time.Time
	Content     string // Combined title + description for analysis
}

// RSS Feed Sources as defined in the plan
var CryptoRSSFeeds = map[string]string{
	"CoinDesk":       "https://www.coindesk.com/arc/outboundfeeds/rss/",
	"CoinTelegraph":  "https://cointelegraph.com/rss",
	"BitcoinCom":     "https://news.bitcoin.com/feed/",
	"Decrypt":        "https://decrypt.co/feed",
	"RedditCrypto":   "https://www.reddit.com/r/cryptocurrency/hot/.rss",
	"RedditCryptoTop": "https://www.reddit.com/r/cryptocurrency/top/.rss?t=day",
}

func NewRSSFeedReader() *RSSFeedReader {
	return &RSSFeedReader{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *RSSFeedReader) FetchFeed(url, source string) ([]NewsItem, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", source, err)
	}
	
	// Set appropriate headers
	req.Header.Set("User-Agent", "CrypGo-Sentiment-Bot/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS from %s: %w", source, err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RSS feed %s returned status %d", source, resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RSS body from %s: %w", source, err)
	}
	
	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse RSS from %s: %w", source, err)
	}
	
	var newsItems []NewsItem
	for _, item := range feed.Channel.Items {
		newsItem := NewsItem{
			Title:       strings.TrimSpace(item.Title),
			Description: strings.TrimSpace(item.Description),
			Link:        strings.TrimSpace(item.Link),
			Source:      source,
			Content:     strings.TrimSpace(item.Title + " " + item.Description),
		}
		
		// Try to parse publication date
		if item.PubDate != "" {
			if pubTime, err := parseRSSDate(item.PubDate); err == nil {
				newsItem.PublishedAt = pubTime
			} else {
				// Fallback to current time if parsing fails
				newsItem.PublishedAt = time.Now()
			}
		} else {
			newsItem.PublishedAt = time.Now()
		}
		
		// Only include items with meaningful content
		if newsItem.Title != "" && newsItem.Content != "" {
			newsItems = append(newsItems, newsItem)
		}
	}
	
	return newsItems, nil
}

// FetchAllFeeds fetches news from all configured RSS sources
func (r *RSSFeedReader) FetchAllFeeds() ([]NewsItem, error) {
	var allNews []NewsItem
	var errors []string
	
	for source, url := range CryptoRSSFeeds {
		newsItems, err := r.FetchFeed(url, source)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", source, err))
			continue
		}
		allNews = append(allNews, newsItems...)
	}
	
	// Return partial results even if some feeds fail
	if len(errors) > 0 && len(allNews) == 0 {
		return nil, fmt.Errorf("all RSS feeds failed: %s", strings.Join(errors, "; "))
	}
	
	return allNews, nil
}

// FetchRecentNews fetches only news from the last N hours
func (r *RSSFeedReader) FetchRecentNews(hoursBack int) ([]NewsItem, error) {
	allNews, err := r.FetchAllFeeds()
	if err != nil {
		return nil, err
	}
	
	cutoff := time.Now().Add(-time.Duration(hoursBack) * time.Hour)
	var recentNews []NewsItem
	
	for _, item := range allNews {
		if item.PublishedAt.After(cutoff) {
			recentNews = append(recentNews, item)
		}
	}
	
	return recentNews, nil
}

// parseRSSDate tries to parse various RSS date formats
func parseRSSDate(dateStr string) (time.Time, error) {
	// Common RSS date formats
	formats := []string{
		time.RFC1123Z, // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,  // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC822Z,  // "02 Jan 06 15:04 -0700"
		time.RFC822,   // "02 Jan 06 15:04 MST"
		"2006-01-02T15:04:05Z07:00", // ISO 8601
		"2006-01-02 15:04:05",       // Simple format
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}