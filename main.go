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
)

func main() {
	scraper := &s.Scraper{}
	pagesData := scraper.DoScrape(sites, max, concurrency, numbers)

	for _, pageData := range pagesData {
		fmt.Println("Site:", pageData.Url)
		for _, word := range pageData.Words {
			fmt.Printf("%s: %d\n", word.Text, word.Count)
		}
	}

	err := scraper.SaveToJson("./data/example_data.json")
	if err != nil {
		log.Println(err)
	}
}
