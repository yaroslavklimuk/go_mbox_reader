package mbox_reader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
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
			parsedId, parsedTime, parsedErr := parseMessagePrefix(tcase.Line)
			fmt.Printf("id: %s time: %s err: %s\n", parsedId, parsedTime, parsedErr)
		})
	}
}

func TestParseMessageHeaders(t *testing.T) {
	type ParseMessageHeadersTestCase struct {
		File    string              `json:"file"`
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
			openedFile, err := os.Open("testcases/distinct-messages/" + tcase.File)
			if err != nil {
				fmt.Println(err)
			}
			msg, err := readMsgContent(openedFile)
			if err != nil {
				fmt.Println(err)
			}
			// fmt.Printf("FCONTENT: %v\n", strings.Join(msg.content, "\n"))
			_, err = parseMessageHeaders(&msg)

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
		MessageFile string              `json:"message-file"`
		Sender      string              `json:"sender"`
		Timestamp   string              `json:"timestamp"`
		Headers     map[string][]string `json:"headers"`
		Bodies      map[string]string   `json:"bodies"`
		Attachments []SectionTestItem   `json:"attachments"`
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

			msg, err := readMsgContent(openedFile)
			if err != nil {
				fmt.Println(err)
			}

			err = parseMessage(&msg)
			if err != nil {
				fmt.Println(err)
			}
			// fmt.Printf("CONTENT: %s\n", strings.Join(msg.content, "\n"))

			// msgSender := string(msg.getSender())
			// if msgSender != tcase.Sender {
			// 	t.Errorf("Sender is not correct. Want:%s, got:%s\n", tcase.Sender, msgSender)
			// }
			// msgTimestamp := msg.getTimestamp().Format(time.RFC3339)
			// if msgTimestamp != tcase.Timestamp {
			// 	t.Errorf("Timestamp is not correct. Want:%s, got:%s\n", tcase.Timestamp, msgTimestamp)
			// }

			// for hkey, hval := range tcase.Headers {
			// 	msgHeader, ok := msg.getHeader(strings.ToUpper(hkey))
			// 	if !ok {
			// 		t.Errorf("Can not find the header %s\n", strings.ToUpper(hkey))
			// 	}
			// 	if len(hval) != len(msgHeader.Values) {
			// 		t.Errorf("Header items count is not correct. Want:%d, got:%d\n", len(hval), len(msgHeader.Values))
			// 	}
			// 	for hind, hitem := range hval {
			// 		msgHitem := string(msgHeader.Values[hind])
			// 		if hitem != msgHitem {
			// 			t.Errorf("%s header value is not correct. Want:%s, got:%s\n", hkey, hitem, msgHitem)
			// 		}
			// 	}
			// }

			// for bkey, tbcontent := range tcase.Bodies {
			// 	bcontent, err := msg.getBody(bkey)
			// 	if err != nil {
			// 		t.Errorf("Error while getting body section %s\n", strings.ToLower(bkey))
			// 	}
			// 	strBContent := string(bcontent)
			// 	if strBContent != tbcontent {
			// 		t.Errorf("%s body content is not correct.\nWant:%s\ngot:%s\n", strings.ToLower(bkey), tbcontent, strBContent)
			// 	}
			// }
		})
	}
}
