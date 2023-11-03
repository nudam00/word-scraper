package main

import (
	"fmt"
	"log"
	s "web-scraper/scraper"
)

var (
	sites = []string{
		"https://websensa.com/en/back-end-developer-2/",
		"https://www.fly4free.com/",
		".xom",
		"https://www.onet.pl/",
		"https://www.wp.pl/",
		"https://spidersweb.pl/",
		"https://www.skyscanner.pl/",
		"https://travelfree.info/",
		"https://www.onet.pl/",
		"https://websensa.com/en/back-end-developer-2/",
	}
	max         = 20
	concurrency = 5
	numbers     = false
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"
	accept      = "text/html"
)

func main() {
	scraper := &s.Scraper{
		Saver:       &s.JsonSaver{},
		Urls:        sites,
		MaxOuput:    max,
		Concurrency: concurrency,
		IfNumIncl:   numbers,
		UserAgent:   userAgent,
		Accept:      accept}

	pagesData := scraper.DoScrape()

	for _, pageData := range pagesData {
		fmt.Println("Site:", pageData.Url)
		for _, word := range pageData.Words {
			fmt.Printf("%s: %d\n", word.Text, word.Count)
		}
	}

	err := scraper.Saver.SaveToJson(pagesData, "./data/example_data.json")
	if err != nil {
		log.Println(err)
	}
}
