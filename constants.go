package mbox_reader

const H_FROM = "FROM"
const H_SUBJECT = "SUBJECT"
const H_DATE = "DATE"
const H_CT_TYPE = "CONTENT-TYPE"
const H_TR_ENC = "CONTENT-TRANSFER-ENCODING"
const H_CT_DISP = "CONTENT-DISPOSITION"
const H_CT_ID = "CONTENT-ID"

const TR_ENC_7BIT = "7bit"
const TR_ENC_QPRNT = "quoted-printable"
const TR_ENC_B64 = "base64"
const TR_ENC_8BIT = "8bit"
const TR_ENC_BIN = "binary"

const CT_MP_MIXED = "multipart/mixed"
const CT_MP_RELATED = "multipart/related"
const CT_MP_ALTER = "multipart/alternative"
const CT_TXT_PLAIN = "text/plain"
const CT_TXT_HTML = "text/html"

const CD_ATTACHMENT = "attachment"
const CD_INLINE = "inline"

const HEAD_TIMESTAMP_FMT = "Mon Jan  2 15:04:05 2006"
