package varstore

// import (
// 	"bytes"
// 	"encoding/binary"
// 	"fmt"
// 	"log"
// 	"os"
// 	"sort"
// 	"strconv"
// 	"strings"

// 	"github.com/appkins-org/go-redfish-uefi/internal/firmware/efi"
// )

// const (
// 	BootOrderName            = "BootOrder"
// 	BootPrefix               = "Boot"
// 	EFI_GLOBAL_VARIABLE_GUID = "8be4df61-93ca-11d2-aa0d-00e098032b8c"
// )

// // GUID constants
// var (
// 	NvDataGUID   = "8d1b55ed-bebf-40b7-8246-d8bd7d64edbe"
// 	FfsGUID      = "8c8ce578-8a3d-4f1c-9935-896185c32dd3"
// 	AuthVarsGUID = "aaf32c78-947b-439a-a180-2e144ec37792"
// )

// // EfiVar represents an EFI variable
// type EfiVar struct {
// 	Name  string
// 	GUID  efi.GUID
// 	Attr  uint32
// 	Data  []byte
// 	Count uint64
// 	PkIdx uint32
// 	Time  [8]byte // Simplified time structure
// }

// // ParseTime parses time data
// func (v *EfiVar) ParseTime(data []byte, offset int) {
// 	copy(v.Time[:], data[offset:offset+8])
// }

// // BytesTime returns time as bytes
// func (v *EfiVar) BytesTime() []byte {
// 	return v.Time[:]
// }

// // EfiVarList is a map of EFI variables
// type EfiVarList map[string]*EfiVar

// // Edk2VarStore represents an EDK2 EFI variable store
// type Edk2VarStore struct {
// 	Filename string
// 	FileData []byte
// 	Start    int
// 	End      int
// 	VarList  *EfiVarList
// }

// // FindNvData finds NvData in the file data
// func FindNvData(data []byte) int {
// 	offset := 0
// 	for offset+64 < len(data) {
// 		guid, err := efi.GUIDFromBytes(data[offset+16:])
// 		if err != nil {
// 			panic(err)
// 		}
// 		// guid := efi.ParseBinGUID(data, offset+16)
// 		// if guid.Equal(efi.NvDataGUID) {
// 		// 	return offset
// 		// }
// 		if guid.Equal(efi.FfsGUID) {
// 			var tlen uint64
// 			var sig uint32
// 			binary.Read(bytes.NewReader(data[offset+32:]), binary.LittleEndian, &tlen)
// 			binary.Read(bytes.NewReader(data[offset+32+8:]), binary.LittleEndian, &sig)
// 			offset += int(tlen)
// 			continue
// 		}
// 		offset += 1024
// 	}
// 	return -1
// }

// // Probe checks if a file is an EDK2 varstore
// func Probe(filename string) bool {
// 	data, err := os.ReadFile(filename)
// 	if err != nil {
// 		return false
// 	}
// 	offset := FindNvData(data)
// 	return offset != -1
// }

// // NewEdk2VarStore creates a new Edk2VarStore from a file
// func NewEdk2VarStore(filename string) (*Edk2VarStore, error) {
// 	vs := &Edk2VarStore{
// 		Filename: filename,
// 	}
// 	err := vs.ReadFile()
// 	if err != nil {
// 		return nil, err
// 	}
// 	err = vs.ParseVolume()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return vs, nil
// }

// // ReadFile reads the raw EDK2 varstore from file
// func (vs *Edk2VarStore) ReadFile() error {
// 	log.Printf("Reading raw edk2 varstore from %s", vs.Filename)
// 	var err error
// 	vs.FileData, err = os.ReadFile(vs.Filename)
// 	return err
// }

// // ParseVolume parses the firmware volume
// func (vs *Edk2VarStore) ParseVolume() error {
// 	offset := FindNvData(vs.FileData)
// 	if offset == -1 {
// 		return fmt.Errorf("%s: varstore not found", vs.Filename)
// 	}

// 	guid := efi.ParseBinGUID(vs.FileData, offset+16)

// 	// Parse volume header
// 	vlen := binary.LittleEndian.Uint32(vs.FileData[offset+32:])
// 	sig := binary.LittleEndian.Uint32(vs.FileData[offset+36:])
// 	_ = binary.LittleEndian.Uint32(vs.FileData[offset+40:])
// 	hlen := binary.LittleEndian.Uint16(vs.FileData[offset+44:])
// 	_ = binary.LittleEndian.Uint16(vs.FileData[offset+46:])
// 	_ = binary.LittleEndian.Uint16(vs.FileData[offset+48:])
// 	rev := vs.FileData[offset+51]
// 	blocks := binary.LittleEndian.Uint32(vs.FileData[offset+52:])
// 	blksize := binary.LittleEndian.Uint32(vs.FileData[offset+56:])

// 	log.Printf("vol=%s vlen=0x%x rev=%d blocks=%d*%d (0x%x)",
// 		guid.String(), vlen, rev, blocks, blksize, blocks*blksize)

// 	if sig != 0x4856465f {
// 		return fmt.Errorf("%s: not a firmware volume", vs.Filename)
// 	}
// 	if guid.String() != NvDataGUID {
// 		return fmt.Errorf("%s: not a variable store", vs.Filename)
// 	}

// 	return vs.ParseVarStore(offset + int(hlen))
// }

// // ParseVarStore parses the variable store
// func (vs *Edk2VarStore) ParseVarStore(start int) error {
// 	guid := efi.ParseBinGUID(vs.FileData, start)

// 	size := binary.LittleEndian.Uint32(vs.FileData[start+16:])
// 	storefmt := vs.FileData[start+20]
// 	state := vs.FileData[start+21]

// 	log.Printf("varstore=%s size=0x%x format=0x%x state=0x%x",
// 		guid.String(), size, storefmt, state)

// 	if guid.String() != AuthVarsGUID {
// 		return fmt.Errorf("%s: unknown varstore guid", vs.Filename)
// 	}
// 	if storefmt != 0x5a {
// 		return fmt.Errorf("%s: unknown varstore format", vs.Filename)
// 	}
// 	if state != 0xfe {
// 		return fmt.Errorf("%s: unknown varstore state", vs.Filename)
// 	}

// 	vs.Start = start + 16 + 12
// 	vs.End = start + int(size)
// 	log.Printf("var store range: 0x%x -> 0x%x", vs.Start, vs.End)
// 	return nil
// }

// // GetVarList gets the list of variables
// func (vs *Edk2VarStore) GetVarList() EfiVarList {
// 	pos := vs.Start
// 	varlist := make(EfiVarList)

// 	for pos < vs.End {
// 		if pos+44 > len(vs.FileData) {
// 			break
// 		}

// 		magic := binary.LittleEndian.Uint16(vs.FileData[pos:])
// 		if magic != 0x55aa {
// 			break
// 		}

// 		state := vs.FileData[pos+2]
// 		attr := binary.LittleEndian.Uint32(vs.FileData[pos+4:])
// 		count := binary.LittleEndian.Uint64(vs.FileData[pos+8:])
// 		pk := binary.LittleEndian.Uint32(vs.FileData[pos+32:])
// 		nsize := binary.LittleEndian.Uint32(vs.FileData[pos+36:])
// 		dsize := binary.LittleEndian.Uint32(vs.FileData[pos+40:])

// 		if state == 0x3f {
// 			varGuid := efi.ParseBinGUID(vs.FileData, pos+44)
// 			name := efi.FromUCS16(vs.FileData, pos+44+16).String()

// 			dataStart := pos + 44 + 16 + int(nsize)
// 			dataEnd := dataStart + int(dsize)
// 			if dataEnd > len(vs.FileData) {
// 				break
// 			}

// 			v := &EfiVar{
// 				Name:  name,
// 				GUID:  varGuid,
// 				Attr:  attr,
// 				Data:  make([]byte, dsize),
// 				Count: count,
// 				PkIdx: pk,
// 			}
// 			copy(v.Data, vs.FileData[dataStart:dataEnd])
// 			v.ParseTime(vs.FileData, pos+16)
// 			varlist[name] = v
// 		}

// 		pos = pos + 44 + 16 + int(nsize) + int(dsize)
// 		// Align to 4 bytes
// 		pos = (pos + 3) & ^3
// 	}

// 	return varlist
// }

// // BytesVar converts a variable to bytes
// func BytesVar(v *EfiVar) []byte {
// 	buf := new(bytes.Buffer)

// 	// Header
// 	binary.Write(buf, binary.LittleEndian, uint16(0x55aa))
// 	buf.WriteByte(0x3f)
// 	buf.WriteByte(0x00) // padding
// 	binary.Write(buf, binary.LittleEndian, v.Attr)
// 	binary.Write(buf, binary.LittleEndian, v.Count)

// 	// Time
// 	buf.Write(v.BytesTime())

// 	name := efi.NewUCS16String(v.Name)

// 	// EfiVar metadata
// 	binary.Write(buf, binary.LittleEndian, v.PkIdx)
// 	binary.Write(buf, binary.LittleEndian, uint32(name.Size()))
// 	binary.Write(buf, binary.LittleEndian, uint32(len(v.Data)))

// 	// GUID and name
// 	buf.Write(v.GUID.BytesLE())
// 	buf.Write(name.Bytes())

// 	// Data
// 	buf.Write(v.Data)

// 	// Align to 4 bytes
// 	padding := (4 - (buf.Len() % 4)) % 4
// 	for i := 0; i < padding; i++ {
// 		buf.WriteByte(0xff)
// 	}

// 	return buf.Bytes()
// }

// // BytesVarList converts a variable list to bytes
// func (vs *Edk2VarStore) BytesVarList(varlist EfiVarList) ([]byte, error) {
// 	buf := new(bytes.Buffer)

// 	// Get sorted keys
// 	keys := make([]string, 0, len(varlist))
// 	for k := range varlist {
// 		keys = append(keys, k)
// 	}
// 	sort.Strings(keys)

// 	// Write variables in sorted order
// 	for _, key := range keys {
// 		buf.Write(BytesVar(varlist[key]))
// 	}

// 	if buf.Len() > vs.End-vs.Start {
// 		return nil, fmt.Errorf("varstore is too small")
// 	}

// 	return buf.Bytes(), nil
// }

// // BytesVarStore converts the entire varstore to bytes
// func (vs *Edk2VarStore) BytesVarStore(varlist EfiVarList) ([]byte, error) {
// 	varBytes, err := vs.BytesVarList(varlist)
// 	if err != nil {
// 		return nil, err
// 	}

// 	buf := new(bytes.Buffer)

// 	// Start portion
// 	buf.Write(vs.FileData[:vs.Start])

// 	// Variables
// 	buf.Write(varBytes)

// 	// Padding
// 	padding := vs.End - vs.Start - buf.Len() + vs.Start
// 	for i := 0; i < padding; i++ {
// 		buf.WriteByte(0xff)
// 	}

// 	// End portion
// 	buf.Write(vs.FileData[vs.End:])

// 	return buf.Bytes(), nil
// }

// // WriteVarStore writes the varstore to a file
// func (vs *Edk2VarStore) WriteVarStore(filename string, varlist EfiVarList) error {
// 	log.Printf("Writing raw edk2 varstore to %s", filename)

// 	blob, err := vs.BytesVarStore(varlist)
// 	if err != nil {
// 		return err
// 	}

// 	return os.WriteFile(filename, blob, 0644)
// }

// // GetBootOrder retrieves the BootOrder variable
// func (vs EfiVarList) GetBootOrder() ([]uint16, error) {

// 	variable, ok := vs["BootOrder"]
// 	if !ok {
// 		return nil, fmt.Errorf("BootOrder variable not found")
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
// func (vs EfiVarList) SetBootOrder(bootOrder []uint16) error {
// 	// Create data
// 	data := make([]byte, len(bootOrder)*2)
// 	for i, id := range bootOrder {
// 		binary.LittleEndian.PutUint16(data[i*2:i*2+2], id)
// 	}

// 	guid, err := efi.ParseGUID(EFI_GLOBAL_VARIABLE_GUID)
// 	if err != nil {
// 		return fmt.Errorf("failed to parse GUID: %v", err)
// 	}

// 	// Create variable
// 	variable := &EfiVar{
// 		Name: BootOrderName,
// 		GUID: guid,
// 		Attr: efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS | efi.EFI_VARIABLE_RUNTIME_ACCESS,
// 		Data: data,
// 	}

// 	vs["BootOrder"] = variable

// 	return nil
// }

// // GetBootEntry retrieves a boot entry by its ID
// func (vs EfiVarList) GetBootEntry(id uint16) (*efi.BootEntry, error) {
// 	name := fmt.Sprintf("%s%04X", BootPrefix, id)

// 	variable, ok := vs[name]
// 	if !ok {
// 		return nil, fmt.Errorf("boot entry not found: %s", name)
// 	}

// 	entry := &efi.BootEntry{}
// 	if err := entry.Parse(variable.Data); err != nil {
// 		return nil, fmt.Errorf("failed to parse boot entry: %v", err)
// 	}

// 	return entry, nil
// }

// // SetBootEntry sets a boot entry
// func (vs EfiVarList) SetBootEntry(id uint16, entry *efi.BootEntry) error {
// 	name := fmt.Sprintf("%s%04X", BootPrefix, id)

// 	// Create variable
// 	variable := &EfiVar{
// 		Name: name,
// 		GUID: efi.EFI_GLOBAL_VARIABLE_GUID,
// 		Attr: efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS | efi.EFI_VARIABLE_RUNTIME_ACCESS,
// 		Data: entry.Bytes(),
// 	}

// 	vs[name] = variable

// 	return nil
// }

// // DeleteBootEntry deletes a boot entry
// func (vs EfiVarList) DeleteBootEntry(id uint16) error {
// 	name := fmt.Sprintf("%s%04X", BootPrefix, id)
// 	delete(vs, name)
// 	return nil
// }

// // ListBootEntries lists all boot entries
// func (vs EfiVarList) ListBootEntries() (map[uint16]*efi.BootEntry, error) {

// 	entries := make(map[uint16]*efi.BootEntry)

// 	for name, v := range vs {
// 		fmt.Printf("EfiVar: %s, GUID: %s, Size: %d\n", name, v.GUID.String(), len(v.Data))

// 		if !strings.HasPrefix(name, BootPrefix) {
// 			continue
// 		}

// 		idStr := strings.TrimPrefix(name, BootPrefix)
// 		id64, err := strconv.ParseUint(idStr, 16, 16)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to parse boot entry ID: %v", err)
// 		}
// 		id := uint16(id64)

// 		entries[id] = &efi.BootEntry{
// 			Attr: v.Attr,
// 		}
// 	}

// 	return entries, nil
// }

// // GetOrderedBootEntries returns boot entries in boot order
// func (vs EfiVarList) GetOrderedBootEntries() ([]*efi.BootEntry, error) {
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
// 	ordered := make([]*efi.BootEntry, 0, len(bootOrder))

// 	for _, id := range bootOrder {
// 		if entry, ok := allEntries[id]; ok {
// 			ordered = append(ordered, entry)
// 		}
// 	}

// 	return ordered, nil
// }
