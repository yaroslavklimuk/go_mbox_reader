package mbox_reader

type AbstractAttachment struct {
	mimeType         string
	transferEncoding string
	content          string
}

type NamedAttachment struct {
	AbstractAttachment
	filename string
	name     string
}

type InlineAttachment struct {
	AbstractAttachment
	contentId string
}

type AbstractAttachmentIface interface {
	getContents(bool) string
	getTransferEncoding() string
	getMimeType() string
}

type NamedAttachmentIface interface {
	AbstractAttachmentIface
	getFileName() string
	getName() string
}

type InlineAttachmentIface interface {
	AbstractAttachmentIface
	getContentId() string
}

func (attachment AbstractAttachment) getContents(decoded bool) string {
	return ""
}

func (attachment AbstractAttachment) getTransferEncoding() string {
	return ""
}

func (attachment AbstractAttachment) getMimeType() string {
	return ""
}

func (attachment NamedAttachment) getFileName() string {
	return ""
}

func (attachment NamedAttachment) getName() string {
	return ""
}

func (attachment InlineAttachment) getContentId() string {
	return ""
}
