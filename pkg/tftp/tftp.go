package tftp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/pin/tftp/v3"
	"github.com/tinkerbell/ipxedust/binary"
)

type Handler struct {
	ctx           context.Context
	RootDirectory string
	Patch         string
}

// ListenAndServe sets up the listener on the given address and serves TFTP requests.
func ListenAndServe(ctx context.Context, addr netip.AddrPort, s *tftp.Server) error {
	a, err := net.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return err
	}

	return Serve(ctx, conn, s)
}

// Serve serves TFTP requests using the given conn and server.
func Serve(_ context.Context, conn net.PacketConn, s *tftp.Server) error {
	return s.Serve(conn)
}

// HandleRead handlers TFTP GET requests. The function signature satisfies the tftp.Server.readHandler parameter type.
func (h *Handler) HandleRead(fullfilepath string, rf io.ReaderFrom) error {
	content, ok := binary.Files[filepath.Base(fullfilepath)]
	if ok {
		return h.HandleIpxeRead(fullfilepath, rf, content)
	}

	root, err := OpenRoot(h.RootDirectory)
	if err != nil {
		log.Errorf("opening root directory %s: %v\n", h.RootDirectory, err)
		return fmt.Errorf("opening root directory %s: %w", h.RootDirectory, err)
	}
	defer root.Close()

	parts := strings.Split(fullfilepath, "/")

	filename := parts[len(parts)-1]
	filedir := strings.Join(parts[:len(parts)-1], "/")
	firstDir := filedir
	if len(parts) > 1 {
		firstDir = parts[0]
	}

	if _, err := net.ParseMAC(firstDir); err == nil {
		rootpath := filename
		if len(parts) > 2 {
			rootpath = strings.Join(parts[1:], "/")
		}

		childExists := false
		if !root.Exists(filedir) {
			log.Infof("creating directories for %s", rootpath)
			// If the mac address directory does not exist, create it.
			err := root.MkdirAll(filedir, 0755)
			if err != nil {
				log.Errorf("creating %s: %v\n", filedir, err)
				return fmt.Errorf("creating %s: %w", filedir, err)
			}
		} else {
			childExists = root.Exists(fullfilepath)
		}

		if root.Exists(rootpath) && !childExists {
			// If the file exists in the new path, but not in the old path, use the new path.
			// This is to support the old path for backwards compatibility.
			newF, err := root.Create(fullfilepath)
			if err != nil {
				log.Errorf("creating %s: %v\n", filename, err)
				return fmt.Errorf("creating %s: %w", filename, err)
			}
			defer newF.Close()
			oldF, err := root.Open(rootpath)
			if err != nil {
				log.Errorf("opening %s: %v\n", rootpath, err)
				return fmt.Errorf("opening %s: %w", rootpath, err)
			}
			defer oldF.Close()
			_, err = io.Copy(newF, oldF)
			if err != nil {
				log.Errorf("copying %s to %s: %v\n", rootpath, filename, err)
				return fmt.Errorf("copying %s to %s: %w", rootpath, filename, err)
			}
		}
	}

	if _, err := root.Stat(fullfilepath); err == nil {
		// file exists
		file, err := root.Open(fullfilepath)
		if err != nil {
			errMsg := fmt.Sprintf("opening %s: %s", fullfilepath, err.Error())
			log.Error(errMsg)
			return errors.New(errMsg)
		}
		n, err := rf.ReadFrom(file)
		if err != nil {
			errMsg := fmt.Sprintf("reading %s: %s", fullfilepath, err.Error())
			log.Error(errMsg)
			return errors.New(errMsg)
		}
		log.Infof("%d bytes sent\n", n)
		return nil

	} else {
		errMsg := fmt.Sprintf("error checking if file exists: %s", fullfilepath)
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	// content, ok := binary.Files[filepath.Base(shortfile)]
	// if !ok {
	// 	err := fmt.Errorf("file [%v] unknown: %w", filepath.Base(shortfile), os.ErrNotExist)
	// 	log.Error(err, "file unknown")
	// 	span.SetStatus(codes.Error, err.Error())
	// 	return err
	// }

	// content, err = binary.Patch(content, t.Patch)
	// if err != nil {
	// 	log.Error(err, "failed to patch binary")
	// 	span.SetStatus(codes.Error, err.Error())
	// 	return err
	// }

	// ct := bytes.NewReader(content)
	// b, err := rf.ReadFrom(ct)
	// if err != nil {
	// 	log.Error(err, "file serve failed", "b", b, "contentSize", len(content))
	// 	span.SetStatus(codes.Error, err.Error())

	// 	return err
	// }
	// log.Info("file served", "bytesSent", b, "contentSize", len(content))
	// span.SetStatus(codes.Ok, filename)

	return nil
}

func (h *Handler) HandleIpxeRead(filename string, rf io.ReaderFrom, content []byte) error {
	content, err := binary.Patch(content, []byte(h.Patch))
	if err != nil {
		log.Error(err, "failed to patch binary")
		return err
	}

	ct := bytes.NewReader(content)
	b, err := rf.ReadFrom(ct)
	if err != nil {
		log.Error(err, "file serve failed", "b", b, "contentSize", len(content))
		return err
	}
	log.Info("file served", "bytesSent", b, "contentSize", len(content))

	return nil
}

// HandleWrite handles TFTP PUT requests. It will always return an error. This library does not support PUT.
func (h *Handler) HandleWrite(filename string, wt io.WriterTo) error {
	root, err := os.OpenRoot(h.RootDirectory)
	if err != nil {
		return fmt.Errorf("opening root directory %s: %w", h.RootDirectory, err)
	}
	defer root.Close()

	file, err := root.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("opening %s: %v\n", filename, err)
		return nil
	}
	n, err := wt.WriteTo(file)
	if err != nil {
		log.Errorf("writing %s: %v\n", filename, err)
		return nil
	}
	log.Infof("%d bytes received", n)
	return nil

	// err := fmt.Errorf("access_violation: %w", os.ErrPermission)
	// client := net.UDPAddr{}
	// if rpi, ok := wt.(tftp.OutgoingTransfer); ok {
	// 	client = rpi.RemoteAddr()
	// }
	// t.Log.Error(err, "client", client, "event", "put", "filename", filename)

	// return err
}
