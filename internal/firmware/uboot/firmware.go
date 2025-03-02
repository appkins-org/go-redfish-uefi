package uboot

import (
	_ "embed"
)

const edk2Conf = `arm_64bit=1
arm_boost=1
enable_uart=1
uart_2ndstage=1
enable_gic=1
kernel=u-boot.bin
# armstub=armstub8.bin
disable_commandline_tags=1
disable_overscan=1
device_tree_address=0x1f0000
device_tree_end=0x200000
dtoverlay=miniuart-bt
dtoverlay=upstream-pi4
tftp_prefix=2
`

const tmp = `enable uart=1
kernel=u-boot.bin
arm 64bit=1
core_freq=250
device tree=bcm2710-rpi-3-b-plus.dtb`

const ubootConf = `boot_ramdisk=1
arm_64bit=1
arm_boost=1
enable_uart=1
kernel=u-boot.bin
device_tree=dt.dtb
tftp_prefix=2
`

// RpiEfi returns the RPI_EFI.fd file.
//
//go:embed RPI_EFI.fd
var RpiEfi []byte

// FixupDat returns the fixup.dat file.
//
//go:embed fixup4.dat
var Fixup4Dat []byte

// Start4ElfDat returns the start4.elf file.
//
//go:embed start4.elf
var Start4ElfDat []byte

// Bcm2711Rpi4BDtb returns the bcm2711-rpi-4-b.dtb file.
//
//go:embed bcm2711-rpi-4-b.dtb
var Bcm2711Rpi4BDtb []byte

// Bcm2711RpiCm4Dtb returns the bcm2711-rpi-cm4.dtb file.
//
//go:embed bcm2711-rpi-cm4.dtb
var Bcm2711RpiCm4Dtb []byte

// Bcm2711Rpi400Dtb returns the bcm2711-rpi-400.dtb file.
//
//go:embed bcm2711-rpi-400.dtb
var Bcm2711Rpi400Dtb []byte

// OverlaysMiniUartBtDtbo returns the overlays/miniuart-bt.dtbo file.
//
//go:embed overlays/miniuart-bt.dtbo
var OverlaysMiniUartBtDtbo []byte

// OverlaysUpstreamPi4Dtbo returns the overlays/upstream-pi4.dtbo file.
//
//go:embed overlays/upstream-pi4.dtbo
var OverlaysUpstreamPi4Dtbo []byte

// FirmwareBrcmBrcmfmac43455SdioBin returns the firmware/brcm/brcmfmac43455-sdio.bin file.
//
//go:embed firmware/brcm/brcmfmac43455-sdio.bin
var FirmwareBrcmBrcmfmac43455SdioBin []byte

// FirmwareBrcmBrcmfmac43455SdioTxt returns the firmware/brcm/brcmfmac43455-sdio.txt file.
//
//go:embed firmware/brcm/brcmfmac43455-sdio.txt
var FirmwareBrcmBrcmfmac43455SdioTxt []byte

// FirmwareBrcmBrcmfmac43455SdioClmBlob returns the firmware/brcm/brcmfmac43455-sdio.clm_blob file.
//
//go:embed firmware/brcm/brcmfmac43455-sdio.clm_blob
var FirmwareBrcmBrcmfmac43455SdioClmBlob []byte

// FirmwareBrcmBrcmfmac43455SdioRaspberry returns the firmware/brcm/brcmfmac43455-sdio.Raspberry file.
//
//go:embed firmware/brcm/brcmfmac43455-sdio.Raspberry
var FirmwareBrcmBrcmfmac43455SdioRaspberry []byte

// UbootBin returns the u-boot.bin file.
//
//go:embed u-boot.bin
var UbootBin []byte

// Initramfs is the default initramfs for the Raspberry Pi 4.
//
//go:embed initramfs
var Initramfs []byte

// ConfigTxt is the default configuration for the Raspberry Pi 4.
var ConfigTxt []byte = []byte(ubootConf)

// DtDtb is the default device tree for the Raspberry Pi 4.
//
//go:embed dt.dtb
var DtDtb []byte

// UbootEnv returns the u-boot.env file.
//
//go:embed u-boot.env
var UbootEnv []byte

// Files is the mapping to the embedded iPXE binaries.
var Files = map[string][]byte{
	"RPI_EFI.fd":                                 RpiEfi,
	"fixup4.dat":                                 Fixup4Dat,
	"start4.elf":                                 Start4ElfDat,
	"bcm2711-rpi-4-b.dtb":                        Bcm2711Rpi4BDtb,
	"bcm2711-rpi-cm4.dtb":                        Bcm2711RpiCm4Dtb,
	"bcm2711-rpi-400.dtb":                        Bcm2711Rpi400Dtb,
	"overlays/miniuart-bt.dtbo":                  OverlaysMiniUartBtDtbo,
	"overlays/upstream-pi4.dtbo":                 OverlaysUpstreamPi4Dtbo,
	"firmware/brcm/brcmfmac43455-sdio.bin":       FirmwareBrcmBrcmfmac43455SdioBin,
	"firmware/brcm/brcmfmac43455-sdio.txt":       FirmwareBrcmBrcmfmac43455SdioTxt,
	"firmware/brcm/brcmfmac43455-sdio.clm_blob":  FirmwareBrcmBrcmfmac43455SdioClmBlob,
	"firmware/brcm/brcmfmac43455-sdio.Raspberry": FirmwareBrcmBrcmfmac43455SdioRaspberry,
	"config.txt":                                 ConfigTxt,
	"u-boot.bin":                                 UbootBin,
	"dt.dtb":                                     DtDtb,
	"initramfs":                                  Initramfs,
	"u-boot.env":                                 UbootEnv,
}
