package scraper

import (
	"encoding/json"
	"os"
)

type JsonSaver struct{}

type DataSaver interface {
	SaveToJson(pagesData []PageData, path string) error
}

// Saves a list of words with a quantity and url to the json file (although gorm can be used to save to a db, for example).
func (js *JsonSaver) SaveToJson(pagesData []PageData, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	err = encoder.Encode(pagesData)
	if err != nil {
		return err
	}

	return nil
}
