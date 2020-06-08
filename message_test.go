package mbox_reader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseMessagePrefix(t *testing.T) {
	type ParseMessagePrefixTestCase struct {
		Line     string `json:"line"`
		Id       string `json:"id"`
		DateTime string `json:"datetime"`
		Error    string `json:"error"`
	}
	testTable := make([]ParseMessagePrefixTestCase, 4)
	data, err := ioutil.ReadFile("testcases/parse_message_prefix_cases.json")
	if err != nil {
		fmt.Println(err)
	}
	json.Unmarshal(data, &testTable)

	for _, tcase := range testTable {
		t.Run(string(tcase.Line), func(t *testing.T) {
			parsedId, parsedTime, parsedErr := ParseMessagePrefix([]byte(tcase.Line))
			fmt.Printf("id: %s time: %s err: %s\n", parsedId, parsedTime, parsedErr)
		})
	}
}

func TestParseMessageHeaders(t *testing.T) {
	type ParseMessageHeadersTestCase struct {
		Text    string              `json:"text"`
		Headers map[string][]string `json:"headers"`
	}
	testTable := make([]ParseMessageHeadersTestCase, 2)
	data, err := ioutil.ReadFile("testcases/parse_headers_cases.json")
	if err != nil {
		fmt.Println(err)
	}
	json.Unmarshal(data, &testTable)

	for ind, tcase := range testTable {
		t.Run(string(ind), func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(tcase.Text))
			msg := &Message{}
			msg.content = make([][]byte, 0)

			err := ParseMessageHeaders(scanner, msg)

			var rHeaders = make(map[string][]string)
			for hkey, hval := range msg.headers {
				rHeaders[hkey] = make([]string, len(hval))
				for hind, hbyte := range hval {
					rHeaders[hkey][hind] = string(hbyte)
				}
			}

			if err != nil {
				t.Error(err)
			} else {
				if !reflect.DeepEqual(rHeaders, tcase.Headers) {
					t.Errorf("want:\n%v\ngot:\n%v\n", tcase.Headers, rHeaders)
				}
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	var err error
	type SectionTestItem struct {
		Headers map[string][]string `json:"headers"`
		Content string              `json:"content"`
	}
	type ParseMessageTestCase struct {
		MessageFile string                     `json:"message-file"`
		Sender      string                     `json:"sender"`
		Timestamp   string                     `json:"timestamp"`
		Headers     map[string][]string        `json:"headers"`
		Bodies      map[string]SectionTestItem `json:"bodies"`
		Attachments []SectionTestItem          `json:"attachments"`
	}
	testTable := make([]ParseMessageTestCase, 1)
	data, err := ioutil.ReadFile("testcases/parse_message_cases.json")
	if err != nil {
		fmt.Println(err)
	}
	json.Unmarshal(data, &testTable)

	for ind, tcase := range testTable {
		t.Run(string(ind), func(t *testing.T) {
			openedFile, err := os.Open("testcases/distinct-messages/" + tcase.MessageFile)
			if err != nil {
				fmt.Println(err)
			}
			reader := bufio.NewReader(openedFile)
			scanner := bufio.NewScanner(reader)
			msg, err := ParseMessage(scanner, reader)
			if err != nil {
				fmt.Println(err)
			}
			msgSender := string(msg.getSender())
			if msgSender != tcase.Sender {
				t.Errorf("Sender is not correct. Want:%s, got:%s\n", tcase.Sender, msgSender)
			}
			msgTimestamp := msg.getTimestamp().Format(time.RFC3339)
			if msgTimestamp != tcase.Timestamp {
				t.Errorf("Timestamp is not correct. Want:%s, got:%s\n", tcase.Timestamp, msgTimestamp)
			}

			for hkey, hval := range tcase.Headers {
				msgHeader := msg.getHeader(strings.ToUpper(hkey))
				fmt.Printf("HEADER: %v\n", msgHeader)
				if hkey != msgHeader.Name {
					t.Errorf("Header name is not correct. Want:%s, got:%s\n", hkey, msgHeader.Name)
				}
				if len(hval) != len(msgHeader.Values) {
					t.Errorf("Header items count is not correct. Want:%d, got:%d\n", len(hval), len(msgHeader.Values))
				}
				// for i, v := range a {
				// 	if v != b[i] {
				// 		return false
				// 	}
				// }
			}
		})
	}
}
