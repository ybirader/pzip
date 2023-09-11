package pzip

import (
	"encoding/binary"
	"time"
)

const extendedTimestampTag = 0x5455

// ExtendedTimeStampExtraField is the extended timestamp field, as defined in the zip specification (See 4.5.3 https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT).
type ExtendedTimestampExtraField struct {
	modified time.Time
}

func NewExtendedTimestampExtraField(modified time.Time) *ExtendedTimestampExtraField {
	return &ExtendedTimestampExtraField{
		modified,
	}
}

// Encode returns the modified time of the associated ExtendedTimestampExtraField as a slice of bytes.
func (e *ExtendedTimestampExtraField) Encode() []byte {
	extraBuf := make([]byte, 0, 9) // 2*SizeOf(uint16) + SizeOf(uint) + SizeOf(uint32)
	extraBuf = binary.LittleEndian.AppendUint16(extraBuf, extendedTimestampTag)
	extraBuf = binary.LittleEndian.AppendUint16(extraBuf, 5) // block size
	extraBuf = append(extraBuf, uint8(1))                    // flags
	extraBuf = binary.LittleEndian.AppendUint32(extraBuf, uint32(e.modified.Unix()))
	return extraBuf
}
