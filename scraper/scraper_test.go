package scraper

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func TestAddPageData(t *testing.T) {
	scraper := Scraper{}
	data := PageData{Url: "abc.com", Words: []Word{
		{Text: "a", Count: 2},
		{Text: "b", Count: 1},
	}}
	scraper.addPageData(data)
	assert.Equal(t, []PageData{data}, scraper.PagesData, "error appending data to pageData")
}

func TestSortWords(t *testing.T) {
	data := map[string]int{
		"a": 2,
		"b": 1,
		"c": 3,
		"d": 15,
		"e": 11,
		"f": 22,
	}

	s := Scraper{MaxOuput: 6}

	words := s.sortWords(data)
	expected := []Word{
		{Text: "f", Count: 22},
		{Text: "d", Count: 15},
		{Text: "e", Count: 11},
		{Text: "c", Count: 3},
		{Text: "a", Count: 2},
		{Text: "b", Count: 1},
	}
	assert.Equal(t, expected, words, "error sorting words")

	s = Scraper{MaxOuput: 1}

	words = s.sortWords(data)
	expected = []Word{
		{Text: "f", Count: 22},
	}
	assert.Equal(t, expected, words, "error sorting words")
}

func TestScrapePage_WrongUrl(t *testing.T) {
	scraper := Scraper{Client: &http.Client{}}
	url := ".com"
	_, err := scraper.scrapePage(url)
	assert.NotNil(t, err)
}

func TestScrapePage(t *testing.T) {
	scraper := Scraper{Client: &http.Client{}, MaxOuput: 20, Re: regexp.MustCompile(`[^\p{L}\p{N} ]+`)}

	url := "https://websensa.com/en/back-end-developer-2/"
	pageData, err := scraper.scrapePage(url)
	assert.Nil(t, err)
	for _, word := range pageData.Words {
		for _, char := range word.Text {
			assert.False(t, unicode.IsPunct(char))
		}
	}
	assert.NotEmpty(t, pageData)

	scraper = Scraper{Client: &http.Client{}, MaxOuput: 20, Re: regexp.MustCompile(`[^\p{L} ]+`)}
	pageData, err = scraper.scrapePage(url)
	assert.Nil(t, err)
	for _, word := range pageData.Words {
		for _, char := range word.Text {
			assert.False(t, unicode.IsPunct(char))
			assert.False(t, unicode.IsDigit(char))
		}
	}
	assert.NotEmpty(t, pageData)
}

func TestDoScrape(t *testing.T) {
	urls := []string{"https://websensa.com/en/back-end-developer-2/"}
	max := 5
	conc := 1
	numb := true
	scraper := Scraper{Urls: urls, MaxOuput: max, Concurrency: conc, IfNumIncl: numb}
	pageData := scraper.DoScrape()
	assert.NotEmpty(t, pageData)

	numb = false
	scraper = Scraper{Urls: urls, MaxOuput: max, Concurrency: conc, IfNumIncl: numb}
	pageData = scraper.DoScrape()
	assert.NotEmpty(t, pageData)
}

func TestDoScrape_DuplicatePage(t *testing.T) {
	urls := []string{"https://websensa.com/en/back-end-developer-2/", "https://websensa.com/en/back-end-developer-2/"}
	max := 5
	conc := 2
	numb := true
	scraper := Scraper{Urls: urls, MaxOuput: max, Concurrency: conc, IfNumIncl: numb}
	pageData := scraper.DoScrape()
	assert.NotEmpty(t, pageData)

	count := 0
	for _, page := range pageData {
		if page.Url == urls[0] {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestDoScrape_WrongPage(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	urls := []string{".com"}
	max := 5
	conc := 1
	numb := true
	scraper := Scraper{Urls: urls, MaxOuput: max, Concurrency: conc, IfNumIncl: numb}
	pageData := scraper.DoScrape()
	assert.Empty(t, pageData)

	assert.Contains(t, buf.String(), string(`Site .com: Get ".com": unsupported protocol scheme ""`))
}

func TestSaveToJson(t *testing.T) {
	f, err := os.CreateTemp("", "test.json")
	assert.Nil(t, err)
	defer os.Remove(f.Name())

	scraper := Scraper{Saver: &JsonSaver{}, PagesData: []PageData{
		{Url: "x.com", Words: []Word{
			{Text: "a", Count: 1},
		}},
	}}
	err = scraper.Saver.SaveToJson([]PageData{
		{Url: "x.com", Words: []Word{
			{Text: "a", Count: 1},
		}}}, f.Name())
	assert.Nil(t, err)

	fileData, err := os.ReadFile(f.Name())
	assert.Nil(t, err)

	var loadedData []PageData
	err = json.Unmarshal(fileData, &loadedData)
	assert.Nil(t, err)

	assert.Equal(t, scraper.PagesData, loadedData)
}

func TestSaveToJson_WrongPath(t *testing.T) {
	scraper := Scraper{Saver: &JsonSaver{}}
	err := scraper.Saver.SaveToJson([]PageData{}, "//X")
	assert.NotNil(t, err)
}
