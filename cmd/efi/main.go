package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/appkins-org/go-redfish-uefi/internal/firmware/efi"
)

func runVirtFwVars(args ...string) (string, error) {
	cmd := exec.Command("virt-fw-vars", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing virt-fw-vars: %v\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

type DevicePathElem struct {
	DevType uint8
	SubType uint8
	Data    []byte
}

// NewDevicePathElem initializes a DevicePathElem from binary data.
func NewDevicePathElem(data []byte) *DevicePathElem {
	if len(data) < 4 {
		return &DevicePathElem{DevType: 0x7F, SubType: 0xFF, Data: []byte{}}
	}

	var devType, subType uint8
	var size uint16

	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &devType)
	binary.Read(buf, binary.LittleEndian, &subType)
	binary.Read(buf, binary.LittleEndian, &size)

	if len(data) < int(size) {
		size = uint16(len(data))
	}
	return &DevicePathElem{DevType: devType, SubType: subType, Data: data[4:size]}
}

// SetIPv4 configures the element as an IPv4 device path.
func (d *DevicePathElem) SetIPv4() {
	d.DevType = 0x03
	d.SubType = 0x0C
	d.Data = make([]byte, 23) // Use DHCP
}

// SetURI configures the element as a URI device path.
func (d *DevicePathElem) SetURI(uri string) {
	d.DevType = 0x03
	d.SubType = 0x18
	d.Data = []byte(uri)
}

// SetFilePath configures the element as a file path device path.
func (d *DevicePathElem) SetFilePath(filepath string) {
	d.DevType = 0x04
	d.SubType = 0x04
	d.Data = []byte(filepath) // Simplified: Real implementation should use UCS-2 encoding
}

// Size returns the size of the device path element.
func (d *DevicePathElem) Size() int {
	return len(d.Data) + 4
}

// Bytes returns the binary representation of the element.
func (d *DevicePathElem) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, d.DevType)
	binary.Write(buf, binary.LittleEndian, d.SubType)
	binary.Write(buf, binary.LittleEndian, uint16(d.Size()))
	buf.Write(d.Data)
	return buf.Bytes()
}

// String returns a human-readable representation of the device path element.
func (d *DevicePathElem) String() string {
	switch d.DevType {
	case 0x01:
		return fmt.Sprintf("HW(subtype=0x%x)", d.SubType)
	case 0x02:
		return fmt.Sprintf("ACPI(subtype=0x%x)", d.SubType)
	case 0x03:
		return fmt.Sprintf("Msg(subtype=0x%x)", d.SubType)
	case 0x04:
		if d.SubType == 0x04 {
			return fmt.Sprintf("FilePath(%s)", string(d.Data))
		}
		return fmt.Sprintf("Media(subtype=0x%x)", d.SubType)
	default:
		return fmt.Sprintf("Unknown(type=0x%x, subtype=0x%x)", d.DevType, d.SubType)
	}
}

// DevicePath represents an EFI device path.
type DevicePath struct {
	Elements []*DevicePathElem
}

// NewDevicePath initializes a DevicePath from binary data.
func NewDevicePath(data []byte) *DevicePath {
	path := &DevicePath{}
	pos := 0
	for pos < len(data) {
		elem := NewDevicePathElem(data[pos:])
		if elem.DevType == 0x7F {
			break
		}
		path.Elements = append(path.Elements, elem)
		pos += elem.Size()
	}
	return path
}

// URI creates a device path with a URI element.
func URI(uri string) *DevicePath {
	path := &DevicePath{}
	elem := &DevicePathElem{}
	elem.SetURI(uri)
	path.Elements = append(path.Elements, elem)
	return path
}

// FilePath creates a device path with a file path element.
func FilePath(filepath string) *DevicePath {
	path := &DevicePath{}
	elem := &DevicePathElem{}
	elem.SetFilePath(filepath)
	path.Elements = append(path.Elements, elem)
	return path
}

// Bytes returns the binary representation of the device path.
func (dp *DevicePath) Bytes() []byte {
	var buf bytes.Buffer
	for _, elem := range dp.Elements {
		buf.Write(elem.Bytes())
	}
	buf.Write(NewDevicePathElem(nil).Bytes()) // End of path element
	return buf.Bytes()
}

// String returns a human-readable representation of the device path.
func (dp *DevicePath) String() string {
	var parts []string
	for _, elem := range dp.Elements {
		parts = append(parts, elem.String())
	}
	return strings.Join(parts, "/")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: edk2-varstore-parser <filename>")
		os.Exit(1)
	}

	filename := os.Args[1]

	vars, err := runVirtFwVars("-i", filename, "--print")
	if err != nil {
		log.Fatal(err)
	}

	dpdata := "010000001800530044002f004d004d00430020006f006e002000410072006100730061006e00200053004400480043004900000001041400fa2c0c1086b598419b4c1683d195b1da7fff04004eac0881119f594d850ee21a522c59b2"

	bootEntry := efi.NewBootEntry([]byte(dpdata), uint32(7), efi.NewUCS16String("netboot grubaa64.efi"), efi.NewURIPath("http://10.0.50.1/grubaa64.efi"), nil)

	bytes, err := json.Marshal(bootEntry)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Boot Entry: %s\n", bootEntry.String())

	fmt.Printf("Device Path: %s\n", bytes)

	fmt.Printf("Variables in %s:\n%s", filename, vars)
}
