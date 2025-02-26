package efi

// import (
// 	"encoding/binary"
// 	"fmt"
// 	"io/ioutil"
// 	"os"
// 	"path/filepath"
// 	"strconv"
// 	"strings"

// 	"github.com/appkins-org/go-redfish-uefi/internal/firmware/varstore"
// )

// // SetVariable writes a variable to the store (continued)
// func (vs *varstore.VarStore) SetVariable(variable *varstore.Variable) error {
// 	filename := filepath.Join(vs.SysfsDir, fmt.Sprintf("%s-%s", variable.Name, variable.Header.GUID.String()))

// 	// Create file data with attributes and variable data
// 	fileData := make([]byte, len(variable.Data)+4)
// 	binary.LittleEndian.PutUint32(fileData[0:4], variable.Attributes)
// 	copy(fileData[4:], variable.Data)

// 	// Write to file
// 	return ioutil.WriteFile(filename, fileData, 0644)
// }

// // DeleteVariable removes a variable from the store
// func (vs *VariableStore) DeleteVariable(name string, guid GUID) error {
// 	filename := filepath.Join(vs.SysfsDir, fmt.Sprintf("%s-%s", name, guid.String()))
// 	return os.Remove(filename)
// }

// // ListVariables lists all variables in the store
// func (vs *VariableStore) ListVariables() ([]Variable, error) {
// 	files, err := ioutil.ReadDir(vs.SysfsDir)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read efivars directory: %v", err)
// 	}

// 	var variables []Variable

// 	for _, file := range files {
// 		// Skip if not a regular file
// 		if !file.Mode().IsRegular() {
// 			continue
// 		}

// 		// Parse filename (format: Name-GUID)
// 		parts := strings.Split(file.Name(), "-")
// 		if len(parts) < 2 {
// 			continue
// 		}

// 		name := parts[0]
// 		guidStr := strings.Join(parts[1:], "-")

// 		guid, err := ParseGUID(guidStr)
// 		if err != nil {
// 			continue
// 		}

// 		// Read variable
// 		variable, err := vs.GetVariable(name, guid)
// 		if err != nil {
// 			continue
// 		}

// 		variables = append(variables, *variable)
// 	}

// 	return variables, nil
// }

// // GetBootOrder retrieves the BootOrder variable
// func (vs *VariableStore) GetBootOrder() ([]uint16, error) {
// 	variable, err := vs.GetVariable(BootOrderName, EFI_GLOBAL_VARIABLE_GUID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Parse boot order (array of uint16)
// 	if len(variable.Data)%2 != 0 {
// 		return nil, fmt.Errorf("invalid boot order data length")
// 	}

// 	numEntries := len(variable.Data) / 2
// 	bootOrder := make([]uint16, numEntries)

// 	for i := 0; i < numEntries; i++ {
// 		bootOrder[i] = binary.LittleEndian.Uint16(variable.Data[i*2 : i*2+2])
// 	}

// 	return bootOrder, nil
// }

// // SetBootOrder sets the BootOrder variable
// func (vs *VariableStore) SetBootOrder(bootOrder []uint16) error {
// 	// Create data
// 	data := make([]byte, len(bootOrder)*2)
// 	for i, id := range bootOrder {
// 		binary.LittleEndian.PutUint16(data[i*2:i*2+2], id)
// 	}

// 	// Create variable
// 	variable := &Variable{
// 		Name:       BootOrderName,
// 		GUID:       EFI_GLOBAL_VARIABLE_GUID,
// 		Attributes: EFI_VARIABLE_NON_VOLATILE | EFI_VARIABLE_BOOTSERVICE_ACCESS | EFI_VARIABLE_RUNTIME_ACCESS,
// 		Data:       data,
// 	}

// 	// Set variable
// 	return vs.SetVariable(variable)
// }

// // GetBootEntry retrieves a boot entry by its ID
// func (vs *VariableStore) GetBootEntry(id uint16) (*BootEntry, error) {
// 	name := fmt.Sprintf("%s%04X", BootPrefix, id)

// 	variable, err := vs.GetVariable(name, EFI_GLOBAL_VARIABLE_GUID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	entry := &BootEntry{}
// 	err = entry.Parse(variable.Data)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse boot entry: %v", err)
// 	}

// 	return entry, nil
// }

// // SetBootEntry sets a boot entry
// func (vs *VariableStore) SetBootEntry(id uint16, entry *BootEntry) error {
// 	name := fmt.Sprintf("%s%04X", BootPrefix, id)

// 	// Create variable
// 	variable := &Variable{
// 		Name:       name,
// 		GUID:       EFI_GLOBAL_VARIABLE_GUID,
// 		Attributes: EFI_VARIABLE_NON_VOLATILE | EFI_VARIABLE_BOOTSERVICE_ACCESS | EFI_VARIABLE_RUNTIME_ACCESS,
// 		Data:       entry.Bytes(),
// 	}

// 	// Set variable
// 	return vs.SetVariable(variable)
// }

// // DeleteBootEntry deletes a boot entry
// func (vs *VariableStore) DeleteBootEntry(id uint16) error {
// 	name := fmt.Sprintf("%s%04X", BootPrefix, id)
// 	return vs.DeleteVariable(name, EFI_GLOBAL_VARIABLE_GUID)
// }

// // ListBootEntries lists all boot entries
// func (vs *VariableStore) ListBootEntries() (map[uint16]*BootEntry, error) {
// 	files, err := ioutil.ReadDir(vs.SysfsDir)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read efivars directory: %v", err)
// 	}

// 	entries := make(map[uint16]*BootEntry)

// 	for _, file := range files {
// 		// Skip if not a regular file
// 		if !file.Mode().IsRegular() {
// 			continue
// 		}

// 		// Check if it's a Boot variable
// 		if !strings.HasPrefix(file.Name(), BootPrefix) {
// 			continue
// 		}

// 		// Parse filename (format: BootXXXX-GUID)
// 		parts := strings.Split(file.Name(), "-")
// 		if len(parts) < 2 || len(parts[0]) < 8 {
// 			continue
// 		}

// 		// Extract boot ID
// 		idStr := parts[0][4:]
// 		id64, err := strconv.ParseUint(idStr, 16, 16)
// 		if err != nil {
// 			continue
// 		}
// 		id := uint16(id64)

// 		// Read boot entry
// 		entry, err := vs.GetBootEntry(id)
// 		if err != nil {
// 			continue
// 		}

// 		entries[id] = entry
// 	}

// 	return entries, nil
// }

// // GetOrderedBootEntries returns boot entries in boot order
// func (vs *VariableStore) GetOrderedBootEntries() ([]*BootEntry, error) {
// 	// Get boot order
// 	bootOrder, err := vs.GetBootOrder()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Get all boot entries
// 	allEntries, err := vs.ListBootEntries()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Create ordered list
// 	ordered := make([]*BootEntry, 0, len(bootOrder))

// 	for _, id := range bootOrder {
// 		if entry, ok := allEntries[id]; ok {
// 			ordered = append(ordered, entry)
// 		}
// 	}

// 	return ordered, nil
// }
