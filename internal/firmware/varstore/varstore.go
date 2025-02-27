package varstore

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/appkins-org/go-redfish-uefi/internal/firmware/efi"
)

const (
	BootOrderName            = "BootOrder"
	BootPrefix               = "Boot"
	EFI_GLOBAL_VARIABLE_GUID = "8be4df61-93ca-11d2-aa0d-00e098032b8c"
)

// GUID constants
var (
	NvDataGUID   = "8d1b55ed-bebf-40b7-8246-d8bd7d64edbe"
	FfsGUID      = "8c8ce578-8a3d-4f1c-9935-896185c32dd3"
	AuthVarsGUID = "aaf32c78-947b-439a-a180-2e144ec37792"
)

// EfiVar represents an EFI variable
type EfiVar struct {
	Name  string   `json:"name,omitempty"`
	GUID  efi.GUID `json:"guid,omitempty"`
	Attr  uint32   `json:"attr,omitempty"`
	Data  []byte   `json:"data,omitempty"`
	Count uint64   `json:"-"`
	PkIdx uint32   `json:"-"`
	Time  [8]byte  `json:"-"`
}

// ParseTime parses time data
func (v *EfiVar) ParseTime(data []byte, offset int) {
	copy(v.Time[:], data[offset:offset+8])
}

// BytesTime returns time as bytes
func (v *EfiVar) BytesTime() []byte {
	return v.Time[:]
}

type fwJson struct {
	Version   int      `json:"version,omitempty"`
	Variables []EfiVar `json:"variables,omitempty"`
}

// EfiVarList is a map of EFI variables
type EfiVarList map[string]*EfiVar

// EfiVariableStore represents an EDK2 EFI variable store
type EfiVariableStore struct {
	Filename string
	VarList  EfiVarList
}

func (vl EfiVarList) ToFwJson() *fwJson {
	vars := make([]EfiVar, 0, len(vl))
	for _, v := range vl {
		vars = append(vars, *v)
	}
	return &fwJson{
		Version:   1,
		Variables: vars,
	}
}

func runVirtFwVars(args ...string) (string, error) {
	cmd := exec.Command("virt-fw-vars", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing virt-fw-vars: %v\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

// NewEfiVariableStore creates a new Edk2VarStore from a file
func NewEfiVariableStore(filename string) (*EfiVariableStore, error) {
	vs := &EfiVariableStore{
		Filename: filename,
	}

	err := vs.ReadFile()
	if err != nil {
		return nil, err
	}
	return vs, nil
}

// ReadFile reads the raw EDK2 varstore from file
func (vs *EfiVariableStore) ReadFile() error {
	log.Printf("Reading raw edk2 varstore from %s", vs.Filename)

	fwFile := vs.getFwFile()
	_, err := runVirtFwVars("-i", vs.Filename, "--output-json", fwFile)
	if err != nil {
		return err
	}

	if exists(fwFile) {
		b, err := os.ReadFile(fwFile)
		if err != nil {
			return err
		}

		fwj := fwJson{}
		err = json.Unmarshal(b, &fwj)
		if err != nil {
			return err
		}

		for _, v := range fwj.Variables {
			vs.VarList[v.Name] = &v
		}
	}

	return nil
}

func (vs *EfiVariableStore) getFwFile() string {
	return strings.Join([]string{path.Dir(vs.Filename), "fw-vars.json"}, string(os.PathSeparator))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return false
}

// GetVarList gets the list of variables
func (vs *EfiVariableStore) GetVarList() EfiVarList {
	return vs.VarList
}

// WriteVarStore writes the varstore to a file
func (vs *EfiVariableStore) WriteVarStore(filename string, varlist EfiVarList) error {

	log.Printf("Writing raw edk2 varstore to %s", filename)

	fwFile := vs.getFwFile()

	f, err := os.Open(fwFile)
	if err != nil {
		return err
	}
	defer f.Close()

	fwj := varlist.ToFwJson()

	b, err := json.Marshal(fwj)
	if err != nil {
		return err
	}

	err = os.WriteFile(fwFile, b, 0755)
	if err != nil {
		return err
	}

	_, err = runVirtFwVars("--inplace", vs.Filename, "--set-json", fwFile)
	if err != nil {
		return err
	}

	vs.VarList = varlist

	return nil

}

// GetBootOrder retrieves the BootOrder variable
func (vs *EfiVariableStore) GetBootOrder() ([]uint16, error) {

	variable, ok := vs.VarList["BootOrder"]
	if !ok {
		return nil, fmt.Errorf("BootOrder variable not found")
	}

	// Parse boot order (array of uint16)
	if len(variable.Data)%2 != 0 {
		return nil, fmt.Errorf("invalid boot order data length")
	}

	numEntries := len(variable.Data) / 2
	bootOrder := make([]uint16, numEntries)

	for i := 0; i < numEntries; i++ {
		bootOrder[i] = binary.LittleEndian.Uint16(variable.Data[i*2 : i*2+2])
	}

	return bootOrder, nil
}

// SetBootOrder sets the BootOrder variable
func (vs *EfiVariableStore) SetBootOrder(bootOrder []uint16) error {
	// Create data
	data := make([]byte, len(bootOrder)*2)
	for i, id := range bootOrder {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], id)
	}

	guid, err := efi.ParseGUID(EFI_GLOBAL_VARIABLE_GUID)
	if err != nil {
		return fmt.Errorf("failed to parse GUID: %v", err)
	}

	// Create variable
	variable := &EfiVar{
		Name: BootOrderName,
		GUID: guid,
		Attr: efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS | efi.EFI_VARIABLE_RUNTIME_ACCESS,
		Data: data,
	}

	vs.VarList["BootOrder"] = variable

	return nil
}

// GetBootEntry retrieves a boot entry by its ID
func (vs *EfiVariableStore) GetBootEntry(id uint16) (*efi.BootEntry, error) {
	name := fmt.Sprintf("%s%04X", BootPrefix, id)

	variable, ok := vs.VarList[name]
	if !ok {
		return nil, fmt.Errorf("boot entry not found: %s", name)
	}

	entry := &efi.BootEntry{}
	if err := entry.Parse(variable.Data); err != nil {
		return nil, fmt.Errorf("failed to parse boot entry: %v", err)
	}

	return entry, nil
}

// SetBootEntry sets a boot entry
func (vs *EfiVariableStore) SetBootEntry(id uint16, entry *efi.BootEntry) error {
	name := fmt.Sprintf("%s%04X", BootPrefix, id)

	// Create variable
	variable := &EfiVar{
		Name: name,
		GUID: efi.EFI_GLOBAL_VARIABLE_GUID,
		Attr: efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS | efi.EFI_VARIABLE_RUNTIME_ACCESS,
		Data: entry.Bytes(),
	}

	vs.VarList[name] = variable

	return nil
}

// DeleteBootEntry deletes a boot entry
func (vs *EfiVariableStore) DeleteBootEntry(id uint16) error {
	name := fmt.Sprintf("%s%04X", BootPrefix, id)
	delete(vs.VarList, name)
	return nil
}

// ListBootEntries lists all boot entries
func (vs *EfiVariableStore) ListBootEntries() (map[uint16]*efi.BootEntry, error) {

	entries := make(map[uint16]*efi.BootEntry)

	for name, v := range vs.VarList {
		fmt.Printf("EfiVar: %s, GUID: %s, Size: %d\n", name, v.GUID.String(), len(v.Data))

		if !strings.HasPrefix(name, BootPrefix) {
			continue
		}

		idStr := strings.TrimPrefix(name, BootPrefix)
		id64, err := strconv.ParseUint(idStr, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse boot entry ID: %v", err)
		}
		id := uint16(id64)

		entries[id] = &efi.BootEntry{
			Attr: v.Attr,
		}
	}

	return entries, nil
}

// GetOrderedBootEntries returns boot entries in boot order
func (vs *EfiVariableStore) GetOrderedBootEntries() ([]*efi.BootEntry, error) {
	// Get boot order
	bootOrder, err := vs.GetBootOrder()
	if err != nil {
		return nil, err
	}

	// Get all boot entries
	allEntries, err := vs.ListBootEntries()
	if err != nil {
		return nil, err
	}

	// Create ordered list
	ordered := make([]*efi.BootEntry, 0, len(bootOrder))

	for _, id := range bootOrder {
		if entry, ok := allEntries[id]; ok {
			ordered = append(ordered, entry)
		}
	}

	return ordered, nil
}
