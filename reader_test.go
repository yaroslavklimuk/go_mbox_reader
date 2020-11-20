package mbox_reader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"
)

func TestReadWithFilters(t *testing.T) {
	type ReadWithFiltersTestCase struct {
		FilePath              string            `json:"filepath"`
		HeaderFilters         map[string]string `json:"header-filters"`
		HeaderRegexFilters    map[string]string `json:"header-regex-filters"`
		FromTime              string            `json:"from-time"`
		BeforeTime            string            `json:"before-time"`
		AttachmentNames       []string          `json:"attachment-names"`
		AttachmentNameRegexes []string          `json:"attachment-name-regex"`
		MessagesFound         uint              `json:"msg-found"`
	}

	testTable := make([]ReadWithFiltersTestCase, 1)
	data, err := ioutil.ReadFile("testcases/reader_read_with_filters_cases.json")
	if err != nil {
		t.Errorf("Couldn't open a file with testcases %e", err)
	}
	json.Unmarshal(data, &testTable)

	for ind, tcase := range testTable {
		t.Run(fmt.Sprint(ind), func(t *testing.T) {
			mboxReader, err := NewMboxReader("testcases/distinct-messages/"+tcase.FilePath, 1, 0)
			if err != nil {
				t.Errorf("Couldn't open the file %e", err)
			}

			fromTime, err := time.Parse(time.RFC1123, tcase.FromTime)
			if err != nil {
				t.Error(err)
			}
			mboxReader.SetAfterTime(fromTime)

			beforeTime, err := time.Parse(time.RFC1123, tcase.BeforeTime)
			if err != nil {
				t.Error(err)
			}
			mboxReader.SetBeforeTime(beforeTime)

			for hkey, headValue := range tcase.HeaderFilters {
				mboxReader.WithHeader(hkey, headValue)
			}
			for hkey, headRgx := range tcase.HeaderRegexFilters {
				mboxReader.WithHeaderRegex(hkey, headRgx)
			}
			for _, attName := range tcase.AttachmentNames {
				mboxReader.WithAttachmentName(attName)
			}
			for _, attRgx := range tcase.AttachmentNameRegexes {
				mboxReader.WithAttachmentNameRegex(attRgx)
			}

			var msgFound uint
			msgFound = 0
			for {
				_, err := mboxReader.Read()
				if err != nil {
					break
				}
				msgFound += 1
			}

			if msgFound != tcase.MessagesFound {
				t.Errorf("Messages count is wrong. Want:%d, got:%d\n", tcase.MessagesFound, msgFound)
			}
		})
	}
}
