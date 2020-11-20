package mbox_reader

import (
	"errors"
	"os"
	"time"

	"github.com/gofrs/flock"
)

type MboxReader struct {
	headerFilters         map[string]string
	headerRegexFilters    map[string]string
	afterTime             time.Time
	beforeTime            time.Time
	attachmentNames       []string
	attachmentNameRegexes []string
	file                  *os.File
	filepath              string
	lockTrialsCount       uint
	lockTrialsTimeout     uint
}

type MboxReaderIface interface {
	Read() (*Message, error)
	setAfterTime(time.Time)
	setBeforeTime(time.Time)
	withHeader(string, string)
	withHeaderRegex(string, string)
	withAttachmentName(string)
	withAttachmentNameRegex(string)
	setFilePath(filepath string) (*MboxReaderIface, error)
	resetFilters()
}

func NewMboxReader(filepath string, lockTrialsCount uint, lockTrialsTimeout uint) (*MboxReader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	mboxReader := &MboxReader{
		file:              file,
		filepath:          filepath,
		lockTrialsCount:   lockTrialsCount,
		lockTrialsTimeout: lockTrialsTimeout,
	}
	mboxReader.headerFilters = make(map[string]string)
	mboxReader.headerRegexFilters = make(map[string]string)

	return mboxReader, nil
}

func (mboxReader *MboxReader) Read() (*Message, error) {
	filelock, err := mboxReader.lockFile()
	if err != nil {
		return nil, err
	}

	defer filelock.Unlock()

	msg := Message{}

	foundMsg := false
	for {
		msg, err = readMsgContent(mboxReader.file)
		if err != nil {
			return nil, err
		}

		err = parseMessage(&msg)
		if err != nil {
			return nil, err
		}

		foundMsg = true

		if !mboxReader.afterTime.IsZero() && mboxReader.afterTime.After(msg.getTimestamp()) {
			foundMsg = false
		}
		if !mboxReader.beforeTime.IsZero() && mboxReader.beforeTime.Before(msg.getTimestamp()) {
			foundMsg = false
		}

		for key, value := range mboxReader.headerFilters {
			msgHeader, ok := msg.getHeader(key)
			if ok && (len(msgHeader.Values) == 0 || msgHeader.Values[0] != value) {
				foundMsg = false
				break
			}
		}

		if foundMsg == true {
			break
		}
	}

	if foundMsg == false {
		err = errors.New("End of file")
	}

	return &msg, err
}

func (mboxReader *MboxReader) lockFile() (filelock *flock.Flock, err error) {
	filelock = flock.New(mboxReader.filepath)
	locked, err := filelock.TryLock()
	var localTrialsCount uint
	if err != nil || locked == false {
		localTrialsCount = 1
		for localTrialsCount <= mboxReader.lockTrialsCount {
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

func (mboxReader *MboxReader) SetAfterTime(afterTime time.Time) *MboxReader {
	mboxReader.afterTime = afterTime
	return mboxReader
}

func (mboxReader *MboxReader) SetBeforeTime(beforeTime time.Time) *MboxReader {
	mboxReader.beforeTime = beforeTime
	return mboxReader
}

func (mboxReader *MboxReader) WithHeader(key string, value string) *MboxReader {
	mboxReader.headerFilters[key] = value
	return mboxReader
}

func (mboxReader *MboxReader) WithHeaderRegex(key string, regex string) *MboxReader {
	mboxReader.headerRegexFilters[key] = regex
	return mboxReader
}

func (mboxReader *MboxReader) WithAttachmentName(name string) *MboxReader {
	mboxReader.attachmentNames = append(mboxReader.attachmentNames, name)
	return mboxReader
}

func (mboxReader *MboxReader) WithAttachmentNameRegex(regex string) *MboxReader {
	mboxReader.attachmentNameRegexes = append(mboxReader.attachmentNameRegexes, regex)
	return mboxReader
}

func (mboxReader *MboxReader) SetFilePath(filepath string) (*MboxReader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	mboxReader.file = file
	mboxReader.filepath = filepath
	return mboxReader, nil
}
