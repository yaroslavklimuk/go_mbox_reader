package mbox_reader

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/quotedprintable"
	"strings"
	"time"
)

type Message struct {
	sender      string
	timestamp   time.Time
	headers     map[string][]string
	bodies      map[string]Section
	attachments []Section
	content     []string
}

type Section struct {
	headers   map[string][]string
	startLine int
	endLine   int
}

type Header struct {
	Name   string
	Values []string
}

type MessageIface interface {
	getSender() string
	getTimestamp() time.Time
	getBody(string) (string, error)
	getHeader(string) (Header, bool)
	getHeaders() []Header
	getAttachments() []AbstractAttachmentIface
	getRawContents() string
}

func (message Message) getSender() string {
	return message.sender
}

func (message Message) getTimestamp() time.Time {
	return message.timestamp
}

func (message Message) getBody(ctype string) (string, error) {
	if currSection, ok := message.bodies[ctype]; ok {
		var rawContent string
		var trasnEncHeader []string
		var transEnc string

		trasnEncHeader, ok = currSection.headers[string(H_TR_ENC)]
		if !ok {
			trasnEncHeader, ok = message.headers[string(H_TR_ENC)]
		}
		if ok && len(trasnEncHeader) > 0 {
			transEnc = trasnEncHeader[0]
		}

		rawContent = strings.Join(message.content[currSection.startLine:currSection.endLine], "")

		if string(transEnc) == string(TR_ENC_QPRNT) {
			decodedContent, err := ioutil.ReadAll(quotedprintable.NewReader(strings.NewReader(rawContent)))
			if err != nil {
				return "", err
			}
			return string(decodedContent), nil

		} else if string(transEnc) == string(TR_ENC_B64) {
			decodedContent, err := base64.StdEncoding.DecodeString(rawContent)
			if err != nil {
				return "", err
			}
			return string(decodedContent), nil
		}

		return rawContent, nil
	}
	return "", nil
}

func (message Message) getHeader(name string) (Header, bool) {
	values, ok := message.headers[name]
	var header Header
	if ok {
		header = Header{
			Name:   name,
			Values: values,
		}
	}
	return header, ok
}

func (message Message) getHeaders() []Header {
	var headers = make([]Header, len(message.headers))
	hind := 0
	for hkey, hvals := range message.headers {
		header := Header{
			Name:   hkey,
			Values: hvals,
		}
		headers[hind] = header
		hind += 1
	}
	return headers
}

func (message Message) getAttachments() []AbstractAttachmentIface {
	return nil
}

func (message Message) getRawContents() string {
	return strings.Join(message.content, "\n")
}

func readMsgContent(reader io.ReadSeeker) (Message, error) {
	scanner := bufio.NewScanner(reader)
	var msg = &Message{}
	var lineStr string

	scanner.Scan()
	lineStr = scanner.Text()
	msg.content = append(msg.content, lineStr)

	for scanner.Scan() {
		lineStr = scanner.Text()
		if reachedNewMessage(lineStr) == true {
			reader.Seek(int64(-len(lineStr)-1), 1)
			break
		}
		msg.content = append(msg.content, lineStr)
	}
	if err := scanner.Err(); err != nil {
		return *msg, err
	}
	return *msg, nil
}

func parseMessage(msg *Message) error {
	id, date, err := parseMessagePrefix(msg.content[0])
	if err != nil {
		return err
	}

	msg.sender = id
	msg.timestamp = date

	var linePos int
	linePos, err = parseMessageHeaders(msg)
	if err != nil {
		return err
	}

	err = parseMessageBody(msg, &linePos)
	if err != nil {
		return err
	}
	return nil
}

func parseMessagePrefix(lineStr string) (id string, date time.Time, err error) {
	const mboxoPrefix = "From "
	const mboxrdPrefix = ">From "

	var prefix string

	if strings.HasPrefix(lineStr, string(mboxoPrefix)) {
		prefix = string(mboxoPrefix)
	} else if strings.HasPrefix(lineStr, string(mboxrdPrefix)) {
		prefix = string(mboxrdPrefix)
	} else {
		err = errors.New("Not a message start line.")
		return
	}

	lineStr = lineStr[len(prefix):]

	idx := strings.Index(lineStr, " ")
	if idx == -1 {
		err = errors.New("A white space after message id is missing.")
		return
	}
	id = lineStr[0:idx]
	lineStr = strings.TrimLeft(lineStr[idx+1:], " ")

	date, err = time.Parse(HEAD_TIMESTAMP_FMT, lineStr)
	if err != nil {
		err = fmt.Errorf("invalid date %q:%q", err.Error(), lineStr)
		return
	}
	return
}

func parseMessageHeaders(msg *Message) (int, error) {
	var linePos int
	linePos = 1
	headers, err := parseHeaders(msg, &linePos)
	msg.headers = headers
	if err != nil {
		return 1, err
	}
	return linePos, nil
}

func parseMessageBody(msg *Message, linePos *int) (err error) {
	isMultipart, boundary, err := messageIsMultipart(msg)
	if err != nil {
		return err
	}
	if isMultipart {
		boundary = "--" + boundary
		err := parseMultipartMessageBody(msg, boundary, linePos)
		if err != nil {
			return err
		}
	} else {
		err := parseSimpleMessageBody(msg, linePos)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseSimpleMessageBody(msg *Message, linePos *int) (err error) {
	var section Section
	section.startLine = *linePos
	newPos := len(msg.content) - 1
	*linePos = newPos
	section.endLine = *linePos
	msg.bodies = make(map[string]Section)
	msg.bodies[string(getMimeTypeFromCType(msg.headers[string(H_CT_TYPE)][0]))] = section
	return nil
}

func parseMultipartMessageBody(msg *Message, boundary string, linePos *int) (err error) {
	for !strings.HasPrefix(msg.content[*linePos], boundary) {
		*linePos += 1
	}
	msg.bodies = make(map[string]Section)
	msg.attachments = make([]Section, 0)
	stopReading := false

	for stopReading == false {
		stopReading, err := parseSection(msg, boundary, linePos)
		if err != nil {
			return err
		}
		if stopReading {
			break
		}
	}
	return nil
}

func parseSection(msg *Message, boundary string, linePos *int) (lastSection bool, err error) {
	*linePos += 1
	sectionHeaders, err := parseHeaders(msg, linePos)
	if err != nil {
		return
	}
	if _, ok := sectionHeaders[string(H_CT_DISP)]; !ok {
		lastSection, err := parseMainContentSection(msg, boundary, sectionHeaders, linePos)
		return lastSection, err
	} else {
		lastSection, err := parseAttachmentSection(msg, boundary, sectionHeaders, linePos)
		return lastSection, err
	}
}

func parseMainContentSection(msg *Message, boundary string,
	sectionHeaders map[string][]string, linePos *int) (lastSection bool, err error) {

	ctype, ok := sectionHeaders[string(H_CT_TYPE)]
	if !ok || ctype[0] == "" {
		err = errors.New("The section does not have a Content-Type header")
		return
	}
	if strings.HasPrefix(ctype[0], string(CT_MP_ALTER)) {
		lastSection, err := parseAlternativeSection(msg, boundary, ctype[0], linePos)
		return lastSection, err
	} else {
		lastSection, err := parseTextSection(msg, boundary, ctype[0], sectionHeaders, linePos)
		return lastSection, err
	}
}

func parseTextSection(msg *Message, boundary string, ctype string,
	sectionHeaders map[string][]string, linePos *int) (lastSection bool, err error) {

	*linePos += 1
	var section Section
	section.startLine = *linePos
	section.headers = sectionHeaders

	for !strings.HasPrefix(msg.content[*linePos], boundary) {
		*linePos += 1
	}
	section.endLine = *linePos
	msg.bodies[string(getMimeTypeFromCType(ctype))] = section

	lastSection = msg.content[*linePos] == boundary+"--"
	return lastSection, err
}

func parseAlternativeSection(msg *Message, boundary string, ctype string, linePos *int) (lastSection bool, err error) {
	alterBoundary := getBoundaryFromCType(ctype)
	// section is corrupted, just read raw content
	if alterBoundary == "" {
		for !strings.HasPrefix(msg.content[*linePos], boundary) {
			*linePos += 1
		}
	} else {
		err := parseMultipartMessageBody(msg, alterBoundary, linePos)
		if err != nil {
			return false, err
		}
		for !strings.HasPrefix(msg.content[*linePos], boundary) {
			*linePos += 1
		}
	}

	lastSection = msg.content[*linePos] == boundary+"--"
	return lastSection, err
}

func parseAttachmentSection(msg *Message, boundary string,
	sectionHeaders map[string][]string, linePos *int) (lastSection bool, err error) {
	var section Section
	section.startLine = *linePos
	section.headers = sectionHeaders
	for !strings.HasPrefix(msg.content[*linePos], boundary) {
		*linePos += 1
	}
	section.endLine = *linePos
	msg.attachments = append(msg.attachments, section)

	lastSection = msg.content[*linePos] == boundary+"--"
	return lastSection, err
}

func parseHeaders(msg *Message, linePos *int) (map[string][]string, error) {
	var currHeaderName string
	var lastHeaderValueIdx = 0
	var headers = make(map[string][]string)

	for len(msg.content[*linePos]) > 0 {
		hname, value, err := parseHeaderLine(msg.content[*linePos])
		if err != nil {
			return nil, err
		}

		if hname != "" {
			if headers[hname] == nil {
				headers[hname] = make([]string, 0)
			}
			headers[hname] = append(headers[hname], value)
			lastHeaderValueIdx = len(headers[hname]) - 1
			currHeaderName = hname
		} else {
			lastValue := headers[currHeaderName][lastHeaderValueIdx]
			lastValue += value
			headers[currHeaderName][lastHeaderValueIdx] = lastValue
		}

		*linePos += 1
	}

	return headers, nil
}
