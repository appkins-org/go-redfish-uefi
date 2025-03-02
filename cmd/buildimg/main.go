package main

import (
	"crypto/rand"
	"log"
	"os"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func unused(_ ...any) {
}

func main() {
	var size int64 = 20 * 1024 * 1024 // 20 MB

	diskImg := "/tmp/disk.img"
	defer os.Remove(diskImg)
	theDisk, _ := diskfs.Create(diskImg, size, diskfs.SectorSizeDefault)

	table := &mbr.Table{
		LogicalSectorSize:  512,
		PhysicalSectorSize: 512,
		Partitions: []*mbr.Partition{
			{
				Bootable: false,
				Type:     mbr.Linux,
				Start:    2048,
				Size:     20480,
			},
		},
	}

	check(theDisk.Partition(table))

	fs, err := theDisk.CreateFilesystem(disk.FilesystemSpec{
		Partition: 1,
		FSType:    filesystem.TypeFat32,
	})
	check(err)

	err = fs.Mkdir("/FOO/BAR")
	check(err)

	rw, err := fs.OpenFile("/FOO/BAR/AFILE.EXE", os.O_CREATE|os.O_RDWR)
	check(err)
	b := make([]byte, 1024)

	_, err = rand.Read(b)
	check(err)

	written, err := rw.Write(b)
	check(err)
	unused(written)
}

func createUboot() {
	var size int64 = 20 * 1024 * 1024 // 20 MB

	diskImg := "/tftpboot/boot.img"
	defer os.Remove(diskImg)
	theDisk, _ := diskfs.Create(diskImg, size, diskfs.SectorSizeDefault)

	fs, err := theDisk.CreateFilesystem(disk.FilesystemSpec{
		Partition: 1,
		FSType:    filesystem.TypeFat32,
	})
	check(err)

	err = fs.Mkdir("/FOO/BAR")
	check(err)

	rw, err := fs.OpenFile("/start4.elf", os.O_CREATE|os.O_RDWR)
	check(err)
	b := make([]byte, 1024)

	_, err = rand.Read(b)
	check(err)

	written, err := rw.Write(b)
	check(err)
	unused(written)
}
