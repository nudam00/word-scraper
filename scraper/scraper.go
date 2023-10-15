package scraper

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/patrickmn/go-cache"
	"golang.org/x/net/html"
)

type Scraper struct {
	PagesData []PageData `json:"pagesData"`
	Cache     *cache.Cache
}

type PageData struct {
	Url   string `json:"url"`
	Words []Word `json:"words"`
}

type Word struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

var (
	re        *regexp.Regexp
	max       int
	client    *http.Client
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"
	accept    = "text/html"
)

// Performs scraping: urls - list of urls, m - maximum number of output words, conc - maximum number of goroutines, num - whether numbers are to be included.
// Goroutines and channels used for concurrency and cache used to not duplicate sites.
func (s *Scraper) DoScrape(urls []string, m int, conc int, num bool) []PageData {
	s.PagesData = make([]PageData, 0, len(urls))
	s.Cache = cache.New(cache.NoExpiration, cache.NoExpiration)
	var wg sync.WaitGroup
	pageDataChan := make(chan PageData, len(urls))
	semaChan := make(chan int, conc)
	client = &http.Client{}

	re = regexp.MustCompile(`[^\p{L}\p{N} ]+`)
	if !num {
		re = regexp.MustCompile(`[^\p{L} ]+`)
	}
	max = m

	for _, url := range urls {
		if _, found := s.Cache.Get(url); found {
			continue
		} else {
			s.Cache.Set(url, true, cache.DefaultExpiration)
		}
		semaChan <- 1
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			defer func() {
				<-semaChan
			}()
			pageData, err := s.scrapePage(url)
			if err != nil {
				log.Printf("Site %s: %s", url, err.Error())
				return
			}
			pageDataChan <- pageData
		}(url)
	}

	go func() {
		wg.Wait()
		close(pageDataChan)
		close(semaChan)
	}()

	for pageData := range pageDataChan {
		s.addPageData(pageData)
	}

	return s.PagesData
}

// Scrapes page, based on given regex - it skips punctuation marks but can also skip numbers with punctuation marks.
// Additional useful things - filtering of some words such as: and, to, i, lub etc. and directing by "href" on a given page but only with the main domain which in this case will allow scraping the whole website. Headers should also be considered - it depends on the site
func (s *Scraper) scrapePage(url string) (PageData, error) {
	wordCounts := make(map[string]int)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return PageData{}, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", accept)

	resp, err := client.Do(req)
	if err != nil {
		return PageData{}, err
	}
	defer resp.Body.Close()

	z := html.NewTokenizer(resp.Body)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() != io.EOF {
				return PageData{}, z.Err()
			}
		case html.TextToken:
			text := string(z.Text())
			words := strings.Fields(text)
			for _, word := range words {
				word = re.ReplaceAllString(word, "")
				if word == "" {
					continue
				}
				word = strings.ToLower(word)
				wordCounts[word]++
			}
		}
		if tt == html.ErrorToken {
			break
		}
	}

	sortedWords := sortWords(wordCounts)
	return PageData{Url: url, Words: sortedWords}, nil
}

// Sorts the words decreasing and leaving a given amount.
func sortWords(wordCounts map[string]int) []Word {
	sortedWords := make([]Word, 0, len(wordCounts))
	for word, count := range wordCounts {
		sortedWords = append(sortedWords, Word{word, count})
	}

	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[j].Count < sortedWords[i].Count
	})

	if len(sortedWords) > max {
		sortedWords = sortedWords[:max]
	}

	return sortedWords
}

// Adds all pagesData together to struct.
func (s *Scraper) addPageData(pageData PageData) {
	s.PagesData = append(s.PagesData, pageData)
}

// Saves a list of words with a quantity and url to the json file (although gorm can be used to save to a db, for example).
func (s *Scraper) SaveToJson(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	err = encoder.Encode(s.PagesData)
	if err != nil {
		return err
	}

	return nil
}
