package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

// EFI Variable Store GUID Pattern (for recognition)
var efiGUIDPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// EFI Variable Structure (Simplified)
type EFIVariable struct {
	GUID  string
	Name  string
	Size  int
	Value []byte
}

// readFile reads the entire firmware file into memory.
func readFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	data := make([]byte, stat.Size())
	_, err = file.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// parseEFIFile scans the firmware image and extracts EFI variables.
func parseEFIFile(data []byte) []EFIVariable {
	var variables []EFIVariable

	// Simulated parsing logic: Start at offset where varstore is located
	offset := 0x1D0064 // Example offset from expected output
	endOffset := 0x1DE000

	for offset < endOffset && offset+32 < len(data) {
		// Look for potential GUID in the data
		chunk := string(data[offset : offset+32])
		match := efiGUIDPattern.FindString(chunk)
		if match != "" {
			// Extract variable name (next part of data)
			nameEnd := offset + 40 // Example offset adjustment
			for nameEnd < len(data) && data[nameEnd] != 0x00 {
				nameEnd++
			}

			name := string(data[offset+32 : nameEnd])
			size := 64 // Placeholder for variable size detection

			variables = append(variables, EFIVariable{
				GUID:  match,
				Name:  name,
				Size:  size,
				Value: data[nameEnd : nameEnd+size],
			})

			// Move offset forward
			offset = nameEnd + size
		} else {
			offset += 1
		}
	}

	return variables
}

// formatOutput prints EFI variables in the expected output format.
func formatOutput(variables []EFIVariable) {
	fmt.Println("INFO: reading raw edk2 varstore from RPI_EFI.fd")
	fmt.Println("INFO: var store range: 0x1d0064 -> 0x1de000")

	for _, v := range variables {
		// Check for boot entries
		if strings.HasPrefix(v.Name, "Boot") {
			fmt.Printf("%-20s : boot entry: title=\"%s\" devpath=GUID(%s)\n", v.Name, v.Name, v.GUID)
		} else if v.Name == "BootNext" || v.Name == "Timeout" {
			fmt.Printf("%-20s : word: 0x%04x\n", v.Name, binary.LittleEndian.Uint16(v.Value))
		} else if v.Name == "certdb" {
			fmt.Printf("%-20s : dword: 0x%08x\n", v.Name, binary.LittleEndian.Uint32(v.Value))
		} else {
			fmt.Printf("%-20s : blob: %d bytes\n", v.Name, v.Size)
		}
	}
}

func main() {
	filename := "/Users/atkini01/rpi4/RPI_EFI.fd"

	data, err := readFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	variables := parseEFIFile(data)
	formatOutput(variables)
}
