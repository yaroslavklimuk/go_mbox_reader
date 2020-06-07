package mbox_reader

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"mime/quotedprintable"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/html/charset"
)

func reachedNewMessage(line []byte) bool {
	if bytes.HasPrefix(line, []byte("From ")) || bytes.HasPrefix(line, []byte("\nFrom ")) ||
		bytes.HasPrefix(line, []byte(">From ")) || bytes.HasPrefix(line, []byte("\n>From ")) {
		return true
	}
	return false
}

func stringIsHeaderName(bSlice []byte) bool {
	badChars := []byte("\r\n\t :")
	badCharsLen := len(badChars)

	for _, item := range bSlice {
		ind := sort.Search(badCharsLen, func(i int) bool {
			return badChars[i] >= item
		})
		if ind < badCharsLen && badChars[ind] == item {
			return false
		}
	}
	return true
}

func parseHeaderLine(line []byte) (name string, value []byte, err error) {
	colonIndex := bytes.Index(line, []byte(":"))
	if colonIndex != -1 {
		hname := line[:colonIndex]
		if stringIsHeaderName(hname) == true {
			name = strings.ToUpper(string(hname))
			value = decodeMimeEncoded(line[colonIndex+1:])
			return
		}
	}

	value = decodeMimeEncoded(line)
	return
}

func isMimeEncoded(line []byte) bool {
	line = bytes.ToLower(line)
	prefixOk := bytes.HasPrefix(line, []byte("=?"))
	suffixOk := bytes.HasSuffix(line, []byte("?="))
	encIdx := bytes.Index(line, []byte("?b?"))
	if encIdx == -1 {
		encIdx = bytes.Index(line, []byte("?q?"))
	}
	return prefixOk == true && suffixOk == true && encIdx != -1 &&
		encIdx > 4 && len(line)-1-encIdx > 4
}

func decodeMimeEncoded(line []byte) []byte {
	items := bytes.Split(line, []byte(" "))
	var resultLine = make([]byte, 0)
	var prevIsMimeEncoded = false
	var err error
	for _, item := range items {
		if isMimeEncoded(item) {
			rgx, _ := regexp.Compile(`([a-zA-Z0-9\-\*]+=\")?=\?([a-zA-Z0-9\-]+)\?([qbQB])\?([a-zA-Z=0-9\"\_\+\-\/\.]+)\?=(\")?`)
			matches := rgx.FindStringSubmatch(string(item))

			prefix := []byte(matches[1])
			charsetEnc := strings.ToLower(matches[2])
			transitEnc := strings.ToLower(matches[3])
			value := matches[4]
			postfix := []byte(matches[5])

			decoded := make([]byte, len(value))
			if transitEnc == "q" {
				decoded, err = ioutil.ReadAll(quotedprintable.NewReader(strings.NewReader(value)))
			} else if transitEnc == "b" {
				decoded, err = base64.StdEncoding.DecodeString(value)
			}

			if err != nil {
				if prevIsMimeEncoded == false {
					resultLine = append(resultLine, []byte(" ")...)
				}
				resultLine = append(resultLine, item...)
				continue
			}

			var convItem = make([]byte, len(decoded))
			encoding, _ := charset.Lookup(charsetEnc)
			convItem, err = encoding.NewDecoder().Bytes(decoded)
			if err != nil {
				if prevIsMimeEncoded == false {
					resultLine = append(resultLine, []byte(" ")...)
				}
				resultLine = append(resultLine, item...)
				continue
			}
			if prevIsMimeEncoded == false {
				resultLine = append(resultLine, []byte(" ")...)
			}
			item = append(prefix, convItem...)
			item = append(item, postfix...)
			resultLine = append(resultLine, item...)
			prevIsMimeEncoded = true
		} else {
			resultLine = append(resultLine, []byte(" ")...)
			resultLine = append(resultLine, item...)
			prevIsMimeEncoded = false
		}
	}
	resultLine = bytes.TrimLeft(resultLine, " ")
	return resultLine
}

func messageIsMultipart(msg *Message) (isMultipart bool, boundary []byte, err error) {
	isMultipart = false

	ctype, ok := msg.headers[string(H_CT_TYPE)]
	if !ok {
		err = errors.New("The message does not have a Content-Type header")
		return
	}

	if bytes.HasPrefix(ctype[0], []byte(CT_MP_MIXED)) || bytes.HasPrefix(ctype[0], []byte(CT_MP_RELATED)) {
		isMultipart = true
		boundary = getBoundaryFromCType(ctype[0])
		if boundary == nil {
			err = errors.New("The message is multipart but a boundary is empty")
			return
		}
	}
	return
}

func getBoundaryFromCType(ctype []byte) []byte {
	r, _ := regexp.Compile(`.+boundary=(\")?([^\"\n]+)(\")?.*`)
	matches := r.FindStringSubmatch(string(ctype))
	if matches[2] != "" {
		return []byte(matches[2])
	}
	return nil
}

func getMimeTypeFromCType(ctype []byte) []byte {
	splitted := bytes.Split(ctype, []byte(";"))
	return bytes.Trim(splitted[0], " \t")
}

func getCharsetFromCType(ctype []byte) []byte {
	r, _ := regexp.Compile(`.+charset=(\")?([^\"\n]+)(\")?.*`)
	matches := r.FindStringSubmatch(string(ctype))
	if matches[2] != "" {
		return []byte(matches[2])
	}
	return nil
}
