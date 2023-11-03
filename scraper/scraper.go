package scraper

import (
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/patrickmn/go-cache"
	"golang.org/x/net/html"
)

type Scraper struct {
	PagesData   []PageData `json:"pagesData"`
	Cache       *cache.Cache
	Saver       DataSaver
	Urls        []string
	MaxOuput    int
	Concurrency int
	IfNumIncl   bool
	Re          *regexp.Regexp
	Client      *http.Client
	UserAgent   string
	Accept      string
}

type PageData struct {
	Url   string `json:"url"`
	Words []Word `json:"words"`
}

type Word struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

// Performs scraping: urls - list of urls, m - maximum number of output words, conc - maximum number of goroutines, num - whether numbers are to be included.
// Goroutines and channels used for concurrency and cache used to not duplicate sites.
func (s *Scraper) DoScrape() []PageData {
	s.PagesData = make([]PageData, 0, len(s.Urls))
	s.Cache = cache.New(cache.NoExpiration, cache.NoExpiration)
	var wg sync.WaitGroup
	pageDataChan := make(chan PageData, len(s.Urls))
	semaChan := make(chan int, s.Concurrency)
	s.Client = &http.Client{}

	// Regex - delete characters other than words and numbers
	s.Re = regexp.MustCompile(`[^\p{L}\p{N} ]+`)
	if !s.IfNumIncl {
		// Regex - delete also numbers
		s.Re = regexp.MustCompile(`[^\p{L} ]+`)
	}

	for _, url := range s.Urls {
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

	req.Header.Set("User-Agent", s.UserAgent)
	req.Header.Set("Accept", s.Accept)

	resp, err := s.Client.Do(req)
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
				word = s.Re.ReplaceAllString(word, "")
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

	sortedWords := s.sortWords(wordCounts)
	return PageData{Url: url, Words: sortedWords}, nil
}

// Sorts the words decreasing and leaving a given amount.
func (s *Scraper) sortWords(wordCounts map[string]int) []Word {
	sortedWords := make([]Word, 0, len(wordCounts))
	for word, count := range wordCounts {
		sortedWords = append(sortedWords, Word{word, count})
	}

	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[j].Count < sortedWords[i].Count
	})

	if len(sortedWords) > s.MaxOuput {
		sortedWords = sortedWords[:s.MaxOuput]
	}

	return sortedWords
}

// Adds all pagesData together to struct.
func (s *Scraper) addPageData(pageData PageData) {
	s.PagesData = append(s.PagesData, pageData)
}
