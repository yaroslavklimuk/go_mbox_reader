package mbox_reader

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"time"
)

type MboxReader struct {
	HeaderFilters         map[string][]byte
	HeaderRegexFilters    map[string][]byte
	FromTime              time.Time
	BeforeTime            time.Time
	AttachmentNames       [][]byte
	AttachmentNameRegexes [][]byte
	reader                io.Reader
}

type MboxReaderIface interface {
	Read() (Message, error)
	setFromTime(time.Time)
	setBeforeTime(time.Time)
	withHeader(string, string)
	withHeaderRegex(string, string)
	withAttachmentName(string)
	withAttachmentNameRegex(string)
	resetFilters()
}

func OpenMboxReader(filepath string) (*MboxReader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	mboxReader := &MboxReader{
		reader: file,
	}

	return mboxReader, nil
}

func (reader *MboxReader) Read() (msg MessageIface, err error) {
	lineScanner := bufio.NewScanner(reader.reader)
	bufReader := bufio.NewReader(reader.reader)
	for {
		msg, err := ParseMessage(lineScanner, bufReader)
		if err != nil {
			return msg, err
		}

		goodMsg := true

		if !reader.FromTime.IsZero() && reader.FromTime.Before(msg.getTimestamp()) {
			goodMsg = false
		}

		if !reader.BeforeTime.IsZero() && reader.FromTime.After(msg.getTimestamp()) {
			goodMsg = false
		}

		for key, value := range reader.HeaderFilters {
			msgHeader, ok := msg.getHeader(key)
			if ok && (len(msgHeader.Values) == 0 || !bytes.Equal(msgHeader.Values[0], value)) {
				goodMsg = false
				break
			}
		}

		if goodMsg == true {
			return msg, nil
		}
	}

	return msg, err
}

func (reader MboxReader) setFromTime(time.Time) {

}

func (reader MboxReader) setBeforeTime(time.Time) {

}

func (reader MboxReader) withHeader(string, string) {

}

func (reader MboxReader) withHeaderRegex(string, string) {

}

func (reader MboxReader) withAttachmentName(string) {

}

func (reader MboxReader) withAttachmentNameRegex(string) {

}

func (reader MboxReader) resetFilters() {

}
