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

func reachedNewMessage(line string) bool {
	return strings.HasPrefix(line, "From ") || strings.HasPrefix(line, ">From ")
}

func stringIsHeaderName(line string) bool {
	bSlice := []byte(line)
	badChars := []byte("\t\n\r :")
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

func parseHeaderLine(line string) (name string, value string, err error) {
	colonIndex := strings.Index(line, ":")
	if colonIndex != -1 {
		hparts := strings.SplitN(line, ":", 2)
		if stringIsHeaderName(hparts[0]) == true {
			name = strings.ToUpper(hparts[0])
			value = decodeMimeEncoded(hparts[1])
			return
		}
	}

	value = decodeMimeEncoded(line)
	return
}

func isMimeEncoded(line string) bool {
	line = strings.ToLower(line)
	prefixOk := strings.HasPrefix(line, "=?")
	suffixOk := strings.HasSuffix(line, "?=")
	encIdx := strings.Index(line, "?b?")
	if encIdx == -1 {
		encIdx = strings.Index(line, "?q?")
	}
	return prefixOk == true && suffixOk == true && encIdx != -1 &&
		encIdx > 4 && len(line)-1-encIdx > 4
}

func decodeMimeEncoded(line string) string {
	items := strings.Split(line, " ")
	var resultLine []byte
	var prevIsMimeEncoded = false
	var err error
	for _, item := range items {
		if isMimeEncoded(item) {
			rgx, _ := regexp.Compile(`([a-zA-Z0-9\-\*]+=\")?=\?([a-zA-Z0-9\-]+)\?([qbQB])\?([a-zA-Z=0-9\"\_\+\-\/\.]+)\?=(\")?`)
			matches := rgx.FindStringSubmatch(item)

			prefix := matches[1]
			charsetEnc := strings.ToLower(matches[2])
			transitEnc := strings.ToLower(matches[3])
			value := matches[4]
			postfix := matches[5]

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
				resultLine = append(resultLine, []byte(item)...)
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

			resultLine = append(resultLine, prefix...)
			resultLine = append(resultLine, convItem...)
			resultLine = append(resultLine, postfix...)
			prevIsMimeEncoded = true
		} else {
			resultLine = append(resultLine, []byte(" ")...)
			resultLine = append(resultLine, item...)
			prevIsMimeEncoded = false
		}
	}
	resultLine = bytes.TrimLeft(resultLine, " ")
	return string(resultLine)
}

func messageIsMultipart(msg *Message) (isMultipart bool, boundary string, err error) {
	isMultipart = false

	ctype, ok := msg.headers[string(H_CT_TYPE)]
	if !ok {
		err = errors.New("The message does not have a Content-Type header")
		return
	}

	if strings.HasPrefix(ctype[0], CT_MP_MIXED) || strings.HasPrefix(ctype[0], CT_MP_RELATED) {
		isMultipart = true
		boundary = getBoundaryFromCType(ctype[0])
		if boundary == "" {
			err = errors.New("The message is multipart but a boundary is empty")
			return
		}
	}
	return
}

func getBoundaryFromCType(ctype string) string {
	r, _ := regexp.Compile(`.+boundary=(\")?([^\"\n]+)(\")?.*`)
	matches := r.FindStringSubmatch(ctype)
	if matches[2] != "" {
		return matches[2]
	}
	return ""
}

func getMimeTypeFromCType(ctype string) string {
	splitted := strings.Split(ctype, ";")
	return strings.Trim(splitted[0], " \t")
}

func getCharsetFromCType(ctype string) string {
	r, _ := regexp.Compile(`.+charset=(\")?([^\"\n]+)(\")?.*`)
	matches := r.FindStringSubmatch(ctype)
	if matches[2] != "" {
		return matches[2]
	}
	return ""
}
