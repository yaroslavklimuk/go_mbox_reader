package mbox_reader

import (
	"io"
	"os"
	"time"

	"github.com/golang/go/src/cmd/go/internal/lockedfile/internal/filelock"
)

type MboxReader struct {
	HeaderFilters         map[string]string
	HeaderRegexFilters    map[string]string
	FromTime              time.Time
	BeforeTime            time.Time
	AttachmentNames       []string
	AttachmentNameRegexes []string
	reader                io.ReadSeeker
}

type MboxReaderIface interface {
	Read() (Message, error)
	setFromTime(time.Time)
	setBeforeTime(time.Time)
	withHeader(string, string)
	withHeaderRegex(string, string)
	withAttachmentName(string)
	withAttachmentNameRegex(string)
	setFilePath(filepath string) error
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
	filelock.Lock(reader.reader)
	defer filelock.Unlock(reader.reader)
	for {
		msg, err := readMsgContent(reader.reader)
		if err != nil {
			return nil, err
		}
		err = parseMessage(&msg)
		if err != nil {
			return nil, err
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
			if ok && (len(msgHeader.Values) == 0 || msgHeader.Values[0] != value) {
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

func (reader *MboxReader) setFromTime(time.Time) {

}

func (reader *MboxReader) setBeforeTime(time.Time) {

}

func (reader *MboxReader) withHeader(string, string) {

}

func (reader *MboxReader) withHeaderRegex(string, string) {

}

func (reader *MboxReader) withAttachmentName(string) {

}

func (reader *MboxReader) withAttachmentNameRegex(string) {

}

func (reader *MboxReader) resetFilters() {

}

func (reader *MboxReader) setFilePath(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}

	reader.reader = file
	return nil
}
