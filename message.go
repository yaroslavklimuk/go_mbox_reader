package mbox_reader

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/quotedprintable"
	"time"
)

type Message struct {
	sender      string
	timestamp   time.Time
	headers     map[string][][]byte
	bodies      map[string]Section
	attachments []Section
	content     [][]byte
}

type Section struct {
	headers   map[string][][]byte
	startLine uint64
	endLine   uint64
}

type Header struct {
	Name   string
	Values [][]byte
}

type MessageIface interface {
	getSender() string
	getTimestamp() time.Time
	getBody(string) ([]byte, error)
	getHeader(string) (Header, bool)
	getHeaders() []Header
	getAttachments() []AbstractAttachmentIface
	getRawContents() []byte
}

func ParseMessage(scanner *bufio.Scanner, reader *bufio.Reader) (*Message, error) {
	msg := &Message{}
	msg.content = make([][]byte, 50)

	var lineBytes []byte
	for scanner.Scan() {
		lineBytes = scanner.Bytes()
		if reachedNewMessage(lineBytes) == true {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	id, date, err := ParseMessagePrefix(lineBytes)
	if err != nil {
		return nil, err
	}
	msg.sender = id
	msg.timestamp = date

	err = ParseMessageHeaders(scanner, msg)
	if err != nil {
		return nil, err
	}

	err = ParseMessageBody(scanner, reader, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func ParseMessagePrefix(lineBytes []byte) (id string, date time.Time, err error) {
	const mboxoPrefix = "From "
	const mboxrdPrefix = ">From "

	var prefix []byte

	if bytes.HasPrefix(lineBytes, []byte(mboxoPrefix)) {
		prefix = []byte(mboxoPrefix)
	} else if bytes.HasPrefix(lineBytes, []byte(mboxrdPrefix)) {
		prefix = []byte(mboxrdPrefix)
	} else {
		err = errors.New("Not a message start line.")
		return
	}

	lineBytes = lineBytes[len(prefix):len(lineBytes)]

	idx := bytes.IndexByte(lineBytes, ' ')
	if idx == -1 {
		err = errors.New("missing ' ' after message id")
		return
	}
	id = string(lineBytes[0:idx])
	lineBytes = bytes.TrimLeft(lineBytes[idx+1:len(lineBytes)], " ")

	date, err = time.Parse(HEAD_TIMESTAMP_FMT, string(lineBytes))
	if err != nil {
		err = fmt.Errorf("invalid date %q:%q", err.Error(), lineBytes)
		return
	}

	return
}

func ParseMessageHeaders(scanner *bufio.Scanner, msg *Message) error {
	headers, err := parseHeaders(scanner, msg)
	msg.headers = headers
	if err != nil {
		return err
	}
	return nil
}

func ParseMessageBody(scanner *bufio.Scanner, reader *bufio.Reader, msg *Message) (err error) {
	isMultipart, boundary, err := messageIsMultipart(msg)
	if err != nil {
		return err
	}
	if isMultipart {
		boundary = append([]byte("--"), boundary...)
		err := ParseMultipartMessageBody(scanner, msg, boundary)
		if err != nil {
			return err
		}
	} else {
		err := ParseSimpleMessageBody(scanner, reader, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func ParseSimpleMessageBody(scanner *bufio.Scanner, reader *bufio.Reader, msg *Message) (err error) {
	var lineBytes []byte
	var section Section
	section.startLine = uint64(len(msg.content))
	for scanner.Scan() {
		lineBytes = scanner.Bytes()
		msg.content = append(msg.content, lineBytes)
		nextBytes, err := reader.Peek(7)
		if err != nil {
			return err
		}

		if reachedNewMessage(nextBytes) {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	section.endLine = uint64(len(msg.content)) - 1
	msg.bodies = make(map[string]Section)
	msg.bodies[string(getMimeTypeFromCType(msg.headers[string(H_CT_TYPE)][0]))] = section
	return nil
}

func ParseMultipartMessageBody(scanner *bufio.Scanner, msg *Message, boundary []byte) (err error) {
	var lineBytes []byte
	for scanner.Scan() {
		lineBytes = scanner.Bytes()
		if bytes.Equal(lineBytes, boundary) {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	msg.bodies = make(map[string]Section)
	msg.attachments = make([]Section, 2)
	stopReading := false

	for stopReading == false {
		stopReading, err := ParseSection(scanner, msg, boundary)
		if err != nil {
			return err
		}
		if stopReading {
			break
		}
	}
	return nil
}

func ParseSection(scanner *bufio.Scanner, msg *Message, boundary []byte) (lastSection bool, err error) {
	sectionHeaders, err := parseHeaders(scanner, msg)
	if err != nil {
		return
	}
	if _, ok := sectionHeaders[string(H_CT_DISP)]; !ok {
		lastSection, err := parseMainContentSection(scanner, msg, boundary, sectionHeaders)
		// fmt.Printf("sec headers %v\n", msg.bodies)
		return lastSection, err
	} else {
		lastSection, err := parseAttachmentSection(scanner, msg, boundary, sectionHeaders)
		return lastSection, err
	}
}

func parseMainContentSection(scanner *bufio.Scanner, msg *Message,
	boundary []byte, sectionHeaders map[string][][]byte) (lastSection bool, err error) {

	ctype, ok := sectionHeaders[string(H_CT_TYPE)]
	if !ok || ctype[0] == nil {
		err = errors.New("The section does not have a Content-Type header")
		return
	}
	if bytes.HasPrefix(ctype[0], []byte(CT_MP_ALTER)) {
		lastSection, err := parseAlternativeSection(scanner, msg, boundary, ctype[0])
		return lastSection, err
	} else {
		lastSection, err := parseTextSection(scanner, msg, boundary, ctype[0], sectionHeaders)
		return lastSection, err
	}
}

func parseTextSection(scanner *bufio.Scanner, msg *Message, boundary []byte,
	ctype []byte, sectionHeaders map[string][][]byte) (lastSection bool, err error) {

	var lineBytes []byte
	var section Section
	section.startLine = uint64(len(msg.content))
	section.headers = sectionHeaders
	for scanner.Scan() {
		lineBytes = scanner.Bytes()
		msg.content = append(msg.content, lineBytes)

		if bytes.HasPrefix(lineBytes, boundary) {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	section.endLine = uint64(len(msg.content)) - 2
	msg.bodies[string(getMimeTypeFromCType(ctype))] = section

	lastBoundary := append(boundary, ([]byte("--"))...)
	if bytes.Equal(lineBytes, lastBoundary) {
		lastSection = true
	} else {
		lastSection = false
	}
	return lastSection, err
}

func parseAlternativeSection(scanner *bufio.Scanner, msg *Message, boundary []byte, ctype []byte) (lastSection bool, err error) {
	var lineBytes []byte
	alterBoundary := getBoundaryFromCType(ctype)
	// section is corrupted, just read raw content
	if alterBoundary == nil {
		for scanner.Scan() {
			lineBytes = scanner.Bytes()
			msg.content = append(msg.content, lineBytes)
			if bytes.HasPrefix(lineBytes, boundary) {
				break
			}
		}
		if err := scanner.Err(); err != nil {
			return false, err
		}
	} else {
		err := ParseMultipartMessageBody(scanner, msg, alterBoundary)
		if err != nil {
			return false, err
		}
		for scanner.Scan() {
			lineBytes = scanner.Bytes()
			msg.content = append(msg.content, lineBytes)
			if bytes.HasPrefix(lineBytes, boundary) {
				break
			}
		}
		if err := scanner.Err(); err != nil {
			return false, err
		}
	}
	lastBoundary := append(boundary, ([]byte("--"))...)
	if bytes.Equal(lineBytes, lastBoundary) {
		lastSection = true
	} else {
		lastSection = false
	}
	return lastSection, err
}

func parseAttachmentSection(scanner *bufio.Scanner, msg *Message,
	boundary []byte, sectionHeaders map[string][][]byte) (lastSection bool, err error) {
	var lineBytes []byte
	var section Section
	section.startLine = uint64(len(msg.content))
	section.headers = sectionHeaders
	for scanner.Scan() {
		lineBytes = scanner.Bytes()
		msg.content = append(msg.content, lineBytes)

		if bytes.HasPrefix(lineBytes, boundary) {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	section.endLine = uint64(len(msg.content)) - 2
	msg.attachments = append(msg.attachments, section)

	lastBoundary := append(boundary, ([]byte("--"))...)
	if bytes.Equal(lineBytes, lastBoundary) {
		lastSection = true
	} else {
		lastSection = false
	}
	return lastSection, err
}

func parseHeaders(scanner *bufio.Scanner, msg *Message) (map[string][][]byte, error) {
	var lineBytes []byte
	var currHeaderName string
	var lastHeaderValueIdx = 0
	var headers = make(map[string][][]byte)
	for scanner.Scan() {
		lineBytes = scanner.Bytes()
		msg.content = append(msg.content, lineBytes)
		//reached an empty line between headers and a body
		if len(lineBytes) == 0 {
			break
		}
		hname, value, err := parseHeaderLine(lineBytes)
		if err != nil {
			return nil, err
		}

		if hname != "" {
			// fmt.Printf("HEADER: %s -- %s\n", hname, string(value))
			if headers[hname] == nil {
				headers[hname] = make([][]byte, 0)
			}
			headers[hname] = append(headers[hname], value)
			lastHeaderValueIdx = len(headers[hname]) - 1
			currHeaderName = hname
		} else {
			// fmt.Printf("HEADER: %s -- %s\n", hname, string(value))
			lastValue := headers[currHeaderName][lastHeaderValueIdx]
			lastValue = append(lastValue, value...)
			headers[currHeaderName][lastHeaderValueIdx] = lastValue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return headers, nil
}

func (message Message) getSender() string {
	return message.sender
}

func (message Message) getTimestamp() time.Time {
	return message.timestamp
}

func (message Message) getBody(ctype string) ([]byte, error) {
	if currSection, ok := message.bodies[ctype]; ok {
		var rawContent []byte
		var trasnEncHeader [][]byte
		var transEnc []byte

		trasnEncHeader, ok = currSection.headers[string(H_TR_ENC)]
		if !ok {
			trasnEncHeader, ok = message.headers[string(H_TR_ENC)]
		}
		if ok && len(trasnEncHeader) > 0 {
			transEnc = trasnEncHeader[0]
		}

		rawContent = bytes.Join(message.content[currSection.startLine:currSection.endLine], []byte(""))

		if string(transEnc) == string(TR_ENC_QPRNT) {
			var decodedContent []byte
			decodedContent, err := ioutil.ReadAll(quotedprintable.NewReader(bytes.NewReader(rawContent)))
			if err != nil {
				return nil, err
			}
			return decodedContent, nil

		} else if string(transEnc) == string(TR_ENC_B64) {
			var decodedContent []byte
			_, err := base64.StdEncoding.Decode(decodedContent, rawContent)
			if err != nil {
				return nil, err
			}
			return decodedContent, nil
		}

		return rawContent, nil
	}
	return nil, nil
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
	return nil
}

func (message Message) getAttachments() []AbstractAttachmentIface {
	return nil
}

func (message Message) getRawContents() []byte {
	return bytes.Join(message.content, []byte("\n"))
}
