package mbox_reader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestParseHeaderLine(t *testing.T) {
	type ParseHeaderLineTestCase struct {
		Line  string `json:"line"`
		Name  string `json:"name"`
		Value string `json:"value"`
		Error string `json:"error"`
	}
	testTable := make([]ParseHeaderLineTestCase, 14)
	data, err := ioutil.ReadFile("testcases/parse_header_line_cases.json")
	if err != nil {
		t.Error(err)
	}
	json.Unmarshal(data, &testTable)

	for ind, tcase := range testTable {
		t.Run(fmt.Sprint(ind), func(t *testing.T) {
			var errText string
			name, value, err := parseHeaderLine(tcase.Line)
			if err != nil {
				errText = err.Error()
			} else {
				errText = ""
			}

			// fmt.Printf("\n\nwant name:%s, value:%s, error:%s\ngot name:%s, value:%s, error:%s\n\n",
			// 	tcase.Name, tcase.Value, tcase.Error, name, string(value), errText)

			if name != tcase.Name || value != tcase.Value || errText != tcase.Error {
				t.Errorf("\nwant name:%s, value:%s, error:%s\ngot name:%s, value:%s, error:%s\n",
					tcase.Name, tcase.Value, tcase.Error, name, value, errText)
			}
		})
	}
}

func TestDecodeMimeEncoded(t *testing.T) {
	type DecodeMimeEncodedTestCase struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	testTable := make([]DecodeMimeEncodedTestCase, 8)
	data, err := ioutil.ReadFile("testcases/decode_mime_encoded_cases.json")
	if err != nil {
		t.Error(err)
	}
	json.Unmarshal(data, &testTable)

	for ind, tcase := range testTable {
		t.Run(fmt.Sprint(ind), func(t *testing.T) {
			output := decodeMimeEncoded(tcase.Input)

			// fmt.Printf("\n\nwant name:%s, value:%s, error:%s\ngot name:%s, value:%s, error:%s\n\n",
			// 	tcase.Name, tcase.Value, tcase.Error, name, string(value), errText)
			tOutput := tcase.Output
			if output != tOutput {
				t.Errorf("\nwant: %v\n got: %v\n", tOutput, output)
			}
		})
	}
}

func TestIsMimeEncoded(t *testing.T) {
	type IsMimeEncodedTestCase struct {
		Input  string `json:"input"`
		Result bool   `json:"result"`
	}
	testTable := make([]IsMimeEncodedTestCase, 5)
	data, err := ioutil.ReadFile("testcases/is_mime_encoded_cases.json")
	if err != nil {
		t.Error(err)
	}
	json.Unmarshal(data, &testTable)

	for ind, tcase := range testTable {
		t.Run(fmt.Sprint(ind), func(t *testing.T) {
			output := isMimeEncoded(tcase.Input)

			// fmt.Printf("\n\nwant name:%s, value:%s, error:%s\ngot name:%s, value:%s, error:%s\n\n",
			// 	tcase.Name, tcase.Value, tcase.Error, name, string(value), errText)

			if output != tcase.Result {
				t.Errorf("\ninput: %s\nwant: %t\ngot: %t\n", tcase.Input, tcase.Result, output)
			}
		})
	}
}
