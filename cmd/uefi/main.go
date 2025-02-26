package main

import (
	"github.com/foxboron/go-uefi/efi"
	"github.com/foxboron/go-uefi/efi/attributes"
	"github.com/foxboron/go-uefi/efi/device"
	"github.com/foxboron/go-uefi/efi/signature"
	"github.com/foxboron/go-uefi/efi/util"
	"github.com/sirupsen/logrus"
)

var (
	cert, _ = util.ReadKeyFromFile("signing.key")
	key, _  = util.ReadCertFromFile("signing.cert")
	sigdata = signature.SignatureData{
		Owner: util.EFIGUID{Data1: 0xc1095e1b, Data2: 0x8a3b, Data3: 0x4cf5, Data4: [8]uint8{0x9d, 0x4a, 0xaf, 0xc7, 0xd7, 0x5d, 0xca, 0x68}},
		Data:  []uint8{}}
)

func main() {

	filename := "/Users/atkini01/rpi4/RPI_EFI.fd"

	a, f, err := attributes.ReadEfivarsFile(filename)

	if err != nil {
		logrus.Fatal(err)
		panic(err)
	}

	logrus.Infof("Attributes: %v", a)

	dpt, err := device.ParseDevicePath(f)
	if err != nil {
		logrus.Fatal(err)
		panic(err)
	}

	efi.GetBootOrder()

	logrus.Infof("Device Path: %v", dpt)
}
