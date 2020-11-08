package mbox_reader

import (
	"errors"
	"os"
	"time"

	"github.com/gofrs/flock"
)

type MboxReader struct {
	HeaderFilters         map[string]string
	HeaderRegexFilters    map[string]string
	FromTime              time.Time
	BeforeTime            time.Time
	AttachmentNames       []string
	AttachmentNameRegexes []string
	filepath              string
	trialsCount           uint
	trialsTimeout         uint
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

func NewMboxReader(filepath string, trialsCount uint, trialsTimeout uint) (*MboxReader, error) {
	mboxReader := &MboxReader{
		filepath:      filepath,
		trialsCount:   trialsCount,
		trialsTimeout: trialsTimeout,
	}

	return mboxReader, nil
}

func (mboxReader *MboxReader) Read() (msg MessageIface, err error) {
	file, filelock, err := mboxReader.openAndLockFile()
	if err != nil {
		return nil, err
	}

	defer filelock.Unlock()
	defer file.Close()

	for {
		msg, err := readMsgContent(file)
		if err != nil {
			return nil, err
		}
		err = parseMessage(&msg)
		if err != nil {
			return nil, err
		}
		goodMsg := true

		if !mboxReader.FromTime.IsZero() && mboxReader.FromTime.Before(msg.getTimestamp()) {
			goodMsg = false
		}
		if !mboxReader.BeforeTime.IsZero() && mboxReader.FromTime.After(msg.getTimestamp()) {
			goodMsg = false
		}

		for key, value := range mboxReader.HeaderFilters {
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

func (mboxReader *MboxReader) openAndLockFile() (lockedFile *os.File, filelock *flock.Flock, err error) {
	file, err := os.Open(mboxReader.filepath)
	if err != nil {
		return nil, nil, err
	}

	filelock, err = mboxReader.lockFile()
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	return file, filelock, nil
}

func (mboxReader *MboxReader) lockFile() (filelock *flock.Flock, err error) {
	filelock = flock.New(mboxReader.filepath)
	locked, err := filelock.TryLock()
	var localTrialsCount uint
	if err != nil || locked == false {
		localTrialsCount = 1
		for localTrialsCount <= mboxReader.trialsCount {
			locked, err := filelock.TryLock()
			localTrialsCount += 1
			if err == nil && locked == true {
				return filelock, nil
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if locked == false {
		return nil, errors.New("Couldn't lock the file")
	}
	return filelock, nil
}

func (reader *MboxReader) SetFromTime(time.Time) {

}

func (reader *MboxReader) SetBeforeTime(time.Time) {

}

func (reader *MboxReader) WithHeader(string, string) {

}

func (reader *MboxReader) WithHeaderRegex(string, string) {

}

func (reader *MboxReader) WithAttachmentName(string) {

}

func (reader *MboxReader) WithAttachmentNameRegex(string) {

}

func (reader *MboxReader) ResetFilters() {

}

func (mboxReader *MboxReader) SetFilePath(filepath string) {
	mboxReader.filepath = filepath
}
