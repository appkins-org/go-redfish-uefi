package efi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// DevicePathElem represents an EFI device path element
type DevicePathElem struct {
	DevType byte
	SubType byte
	Data    []byte
}

// DevicePath represents a complete EFI device path consisting of multiple elements
type DevicePath struct {
	Elements []DevicePathElem
}

// NewDevicePathElem creates a new device path element from binary data
func NewDevicePathElem(data []byte) DevicePathElem {
	elem := DevicePathElem{
		DevType: 0x7f,
		SubType: 0xff,
		Data:    []byte{},
	}

	if len(data) >= 4 {
		elem.DevType = data[0]
		elem.SubType = data[1]
		size := binary.LittleEndian.Uint16(data[2:4])
		if len(data) >= int(size) && size >= 4 {
			elem.Data = data[4:size]
		}
	}

	return elem
}

// SetIPv4 sets the element to IPv4 type with DHCP settings
func (e *DevicePathElem) SetIPv4() {
	e.DevType = 0x03          // msg
	e.SubType = 0x0c          // ipv4
	e.Data = make([]byte, 23) // use dhcp
}

// SetURI sets the element to URI type with the given URI
func (e *DevicePathElem) SetURI(uri string) {
	e.DevType = 0x03 // msg
	e.SubType = 0x18 // uri
	e.Data = []byte(uri)
}

// SetFilepath sets the element to filepath type with the given path
func (e *DevicePathElem) SetFilepath(filepath string) {
	e.DevType = 0x04 // media
	e.SubType = 0x04 // filepath
	e.Data = FromString(filepath).Bytes()
}

// SetGPT sets the element to GPT partition data
func (e *DevicePathElem) SetGPT(pnr uint32, poff uint64, plen uint64, guid string) {
	e.DevType = 0x04 // media
	e.SubType = 0x01 // hard drive

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, pnr)
	binary.Write(buf, binary.LittleEndian, poff)
	binary.Write(buf, binary.LittleEndian, plen)

	guidBytes, _ := ParseGUIDString(guid)
	buf.Write(guidBytes)

	buf.WriteByte(0x02) // GPT
	buf.WriteByte(0x02) // signature type

	e.Data = buf.Bytes()
}

// FmtHW formats hardware device paths
func (e *DevicePathElem) FmtHW() string {
	if e.SubType == 0x01 && len(e.Data) >= 2 {
		func_ := e.Data[0]
		dev := e.Data[1]
		return fmt.Sprintf("PCI(dev=%02x:%x)", dev, func_)
	}
	if e.SubType == 0x04 && len(e.Data) >= 16 {
		guid := FormatGUID(e.Data[0:16])
		return fmt.Sprintf("VendorHW(%s)", guid)
	}
	return fmt.Sprintf("HW(subtype=0x%x)", e.SubType)
}

// FmtACPI formats ACPI device paths
func (e *DevicePathElem) FmtACPI() string {
	if e.SubType == 0x01 && len(e.Data) >= 8 {
		hid := binary.LittleEndian.Uint32(e.Data[0:4])
		uid := binary.LittleEndian.Uint32(e.Data[4:8])
		if hid == 0xa0341d0 {
			return "PciRoot()"
		}
		return fmt.Sprintf("ACPI(hid=0x%x,uid=0x%x)", hid, uid)
	}
	if e.SubType == 0x03 && len(e.Data) >= 4 {
		adr := binary.LittleEndian.Uint32(e.Data[0:4])
		return fmt.Sprintf("GOP(adr=0x%x)", adr)
	}
	return fmt.Sprintf("ACPI(subtype=0x%x)", e.SubType)
}

// FmtMsg formats message device paths
func (e *DevicePathElem) FmtMsg() string {
	if e.SubType == 0x02 && len(e.Data) >= 4 {
		pun := binary.LittleEndian.Uint16(e.Data[0:2])
		lun := binary.LittleEndian.Uint16(e.Data[2:4])
		return fmt.Sprintf("SCSI(pun=%d,lun=%d)", pun, lun)
	}
	if e.SubType == 0x05 && len(e.Data) >= 2 {
		port := e.Data[0]
		//intf := e.Data[1]
		return fmt.Sprintf("USB(port=%d)", port)
	}
	if e.SubType == 0x0b {
		return "MAC()"
	}
	if e.SubType == 0x0c {
		return "IPv4()"
	}
	if e.SubType == 0x0d {
		return "IPv6()"
	}
	if e.SubType == 0x12 && len(e.Data) >= 6 {
		port := binary.LittleEndian.Uint16(e.Data[0:2])
		//mul := binary.LittleEndian.Uint16(e.Data[2:4])
		//lun := binary.LittleEndian.Uint16(e.Data[4:6])
		return fmt.Sprintf("SATA(port=%d)", port)
	}
	if e.SubType == 0x13 && len(e.Data) >= 14 {
		//proto := binary.LittleEndian.Uint16(e.Data[0:2])
		//login := binary.LittleEndian.Uint16(e.Data[2:4])
		//lun := binary.LittleEndian.Uint64(e.Data[4:12])
		//tag := binary.LittleEndian.Uint16(e.Data[12:14])
		target := string(e.Data[14:])
		return fmt.Sprintf("ISCSI(%s)", target)
	}
	if e.SubType == 0x18 {
		return fmt.Sprintf("URI(%s)", string(e.Data))
	}
	if e.SubType == 0x1f {
		return "DNS()"
	}
	return fmt.Sprintf("Msg(subtype=0x%x)", e.SubType)
}

// FmtMedia formats media device paths
func (e *DevicePathElem) FmtMedia() string {
	if e.SubType == 0x01 && len(e.Data) >= 20 {
		pnr := binary.LittleEndian.Uint32(e.Data[0:4])
		//pstart := binary.LittleEndian.Uint64(e.Data[4:12])
		//pend := binary.LittleEndian.Uint64(e.Data[12:20])
		return fmt.Sprintf("Partition(nr=%d)", pnr)
	}
	if e.SubType == 0x04 {
		path := FromUCS16(e.Data)
		return fmt.Sprintf("FilePath(%s)", path)
	}
	if e.SubType == 0x06 && len(e.Data) >= 16 {
		guid := FormatGUID(e.Data[0:16])
		return fmt.Sprintf("FvFileName(%s)", guid)
	}
	if e.SubType == 0x07 && len(e.Data) >= 16 {
		guid := FormatGUID(e.Data[0:16])
		return fmt.Sprintf("FvName(%s)", guid)
	}
	return fmt.Sprintf("Media(subtype=0x%x)", e.SubType)
}

// Size returns the size of the device path element
func (e *DevicePathElem) Size() int {
	return len(e.Data) + 4
}

// Bytes converts the device path element to its binary representation
func (e *DevicePathElem) Bytes() []byte {
	size := uint16(e.Size())
	buf := new(bytes.Buffer)
	buf.WriteByte(e.DevType)
	buf.WriteByte(e.SubType)
	binary.Write(buf, binary.LittleEndian, size)
	buf.Write(e.Data)
	return buf.Bytes()
}

// String returns a string representation of the device path element
func (e *DevicePathElem) String() string {
	switch e.DevType {
	case 0x01:
		return e.FmtHW()
	case 0x02:
		return e.FmtACPI()
	case 0x03:
		return e.FmtMsg()
	case 0x04:
		return e.FmtMedia()
	default:
		return fmt.Sprintf("Unknown(type=0x%x,subtype=0x%x)", e.DevType, e.SubType)
	}
}

// Equals checks if two device path elements are equal
func (e *DevicePathElem) Equals(other DevicePathElem) bool {
	if e.DevType != other.DevType {
		return false
	}
	if e.SubType != other.SubType {
		return false
	}

	if e.DevType == 0x04 && e.SubType == 0x04 {
		// FilePath -> compare case-insensitive
		p1 := strings.ToLower(FromUCS16(e.Data).String())
		p2 := strings.ToLower(FromUCS16(other.Data).String())
		return p1 == p2
	}

	return bytes.Equal(e.Data, other.Data)
}

// NewDevicePath creates a new device path from binary data
func NewDevicePath(data []byte) DevicePath {
	path := DevicePath{
		Elements: []DevicePathElem{},
	}

	if len(data) == 0 {
		return path
	}

	pos := 0
	for pos < len(data) {
		elem := NewDevicePathElem(data[pos:])
		if elem.DevType == 0x7f {
			break
		}
		path.Elements = append(path.Elements, elem)
		pos += elem.Size()
	}

	return path
}

// NewURIPath creates a new device path for a URI
func NewURIPath(uri string) DevicePath {
	path := DevicePath{}
	elem := DevicePathElem{}
	elem.SetURI(uri)
	path.Elements = append(path.Elements, elem)
	return path
}

// NewFilePath creates a new device path for a file path
func NewFilePath(filepath string) DevicePath {
	path := DevicePath{}
	elem := DevicePathElem{}
	elem.SetFilepath(filepath)
	path.Elements = append(path.Elements, elem)
	return path
}

// Bytes converts the device path to its binary representation
func (p *DevicePath) Bytes() []byte {
	buf := new(bytes.Buffer)
	for _, elem := range p.Elements {
		buf.Write(elem.Bytes())
	}

	// End of device path
	endElem := DevicePathElem{
		DevType: 0x7f,
		SubType: 0xff,
		Data:    []byte{},
	}
	buf.Write(endElem.Bytes())

	return buf.Bytes()
}

// String returns a string representation of the device path
func (p *DevicePath) String() string {
	parts := make([]string, len(p.Elements))
	for i, elem := range p.Elements {
		parts[i] = elem.String()
	}
	return strings.Join(parts, "/")
}

// Equals checks if two device paths are equal
func (p *DevicePath) Equals(other DevicePath) bool {
	if len(p.Elements) != len(other.Elements) {
		return false
	}

	for i := 0; i < len(p.Elements); i++ {
		if !p.Elements[i].Equals(other.Elements[i]) {
			return false
		}
	}

	return true
}

// ParseGUIDString parses a GUID string into bytes in little-endian format
func ParseGUIDString(guid string) ([]byte, error) {
	// Simple placeholder for GUID parsing
	// In a real implementation, you would properly parse the GUID string format
	// This is a simplified version that just creates some placeholder bytes
	return make([]byte, 16), nil
}

// FormatGUID formats a GUID byte array to string
func FormatGUID(data []byte) string {
	if len(data) < 16 {
		return "invalid-guid"
	}

	// Simple placeholder for GUID formatting
	// In a real implementation, you would properly format the GUID according to standard
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		data[0:4], data[4:6], data[6:8], data[8:10], data[10:16])
}
