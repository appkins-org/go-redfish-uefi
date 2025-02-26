package efi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// GUID represents an EFI GUID (Globally Unique Identifier)
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

func (g GUID) BytesLE() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, g.Data1)
	binary.Write(buf, binary.LittleEndian, g.Data2)
	binary.Write(buf, binary.LittleEndian, g.Data3)
	buf.Write(g.Data4[:])
	return buf.Bytes()
}

// ParseGUID parses a GUID from its string representation
func ParseGUID(s string) (GUID, error) {
	var guid GUID

	// Remove braces and whitespace
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")

	// Check length
	if len(s) != 32 {
		return guid, fmt.Errorf("invalid GUID string length: %d", len(s))
	}

	// Parse the four parts
	var err error

	data1, err := strconv.ParseUint(s[0:8], 16, 32)
	if err != nil {
		return guid, fmt.Errorf("failed to parse Data1: %v", err)
	}
	guid.Data1 = uint32(data1)

	data2, err := strconv.ParseUint(s[8:12], 16, 16)
	if err != nil {
		return guid, fmt.Errorf("failed to parse Data2: %v", err)
	}
	guid.Data2 = uint16(data2)

	data3, err := strconv.ParseUint(s[12:16], 16, 16)
	if err != nil {
		return guid, fmt.Errorf("failed to parse Data3: %v", err)
	}
	guid.Data3 = uint16(data3)

	for i := 0; i < 8; i++ {
		val, err := strconv.ParseUint(s[16+i*2:18+i*2], 16, 8)
		if err != nil {
			return guid, fmt.Errorf("failed to parse Data4[%d]: %v", i, err)
		}
		guid.Data4[i] = byte(val)
	}

	return guid, nil
}

// NewGUID creates a new GUID from its components
func NewGUID(data1 uint32, data2, data3 uint16, data4 [8]byte) GUID {
	return GUID{
		Data1: data1,
		Data2: data2,
		Data3: data3,
		Data4: data4,
	}
}

// FromBytes parses a GUID from its binary representation
func GUIDFromBytes(data []byte) (GUID, error) {
	if len(data) < 16 {
		return GUID{}, fmt.Errorf("data too short for GUID, need 16 bytes")
	}

	var guid GUID
	guid.Data1 = binary.LittleEndian.Uint32(data[0:4])
	guid.Data2 = binary.LittleEndian.Uint16(data[4:6])
	guid.Data3 = binary.LittleEndian.Uint16(data[6:8])
	copy(guid.Data4[:], data[8:16])

	return guid, nil
}

// Bytes returns the binary representation of the GUID
func (g GUID) Bytes() []byte {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint32(data[0:4], g.Data1)
	binary.LittleEndian.PutUint16(data[4:6], g.Data2)
	binary.LittleEndian.PutUint16(data[6:8], g.Data3)
	copy(data[8:16], g.Data4[:])
	return data
}

// String returns the standard string representation of the GUID
func (g GUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3],
		g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

// ParseBinGUID parses a binary GUID from data at offset
func ParseBinGUID(data []byte, offset int) GUID {
	var guid GUID

	guid.Data1 = binary.LittleEndian.Uint32(data[offset : offset+4])
	guid.Data2 = binary.LittleEndian.Uint16(data[offset+4 : offset+6])
	guid.Data3 = binary.LittleEndian.Uint16(data[offset+6 : offset+8])
	guid.Data4 = [8]byte{
		data[offset+8],
		data[offset+9],
		data[offset+10],
		data[offset+11],
		data[offset+12],
		data[offset+13],
		data[offset+14],
		data[offset+15],
	}
	return guid
}

// Equal compares two GUIDs for equality
func (g GUID) Equal(other GUID) bool {
	return g.Data1 == other.Data1 &&
		g.Data2 == other.Data2 &&
		g.Data3 == other.Data3 &&
		g.Data4 == other.Data4
}

// Common EFI GUIDs
var (
	EFI_GLOBAL_VARIABLE_GUID    = GUID{0x8BE4DF61, 0x93CA, 0x11d2, [8]byte{0xAA, 0x0D, 0x00, 0xE0, 0x98, 0x03, 0x2B, 0x8C}}
	EFI_IMAGE_SECURITY_DATABASE = GUID{0xd719b2cb, 0x3d3a, 0x4596, [8]byte{0xa3, 0xbc, 0xda, 0xd0, 0x0e, 0x67, 0x65, 0x6f}}
	MICROSOFT_GUID              = GUID{0x77fa9abd, 0x0359, 0x4d32, [8]byte{0xbd, 0x60, 0x28, 0xf4, 0xe7, 0x8f, 0x78, 0x4b}}
	NvDataGUID                  = GUID{0x8d1b55ed, 0xbebf, 0x40b7, [8]byte{0x82, 0x46, 0xd8, 0xbd, 0x7d, 0x64, 0xed, 0xbe}}
	FfsGUID                     = GUID{0x8c8ce578, 0x8a3d, 0x4f1c, [8]byte{0x99, 0x35, 0x89, 0x61, 0x85, 0xc3, 0x2d, 0xd3}}
	AuthVarsGUID                = GUID{0xaaf32c78, 0x947b, 0x439a, [8]byte{0xa1, 0x80, 0x2e, 0x14, 0x4e, 0xc3, 0x77, 0x92}}
)
