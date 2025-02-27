package tftp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/appkins-org/go-redfish-uefi/internal/firmware/uboot"
	"github.com/go-logr/logr"

	"github.com/pin/tftp/v3"
	"github.com/tinkerbell/ipxedust/binary"
)

type Server struct {
	Logger        logr.Logger
	RootDirectory string
	Patch         string
	Log           logr.Logger
}

type Handler struct {
	ctx           context.Context
	RootDirectory string
	Patch         string
	Log           logr.Logger
}

// ListenAndServe sets up the listener on the given address and serves TFTP requests.
func (r *Server) ListenAndServe(ctx context.Context, addr netip.AddrPort) error {
	tftpHandler := &Handler{
		ctx:           ctx,
		RootDirectory: r.RootDirectory,
		Patch:         r.Patch,
		Log:           r.Logger,
	}

	s := tftp.NewServer(tftpHandler.HandleRead, tftpHandler.HandleWrite)

	a, err := net.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		r.Logger.Info("shutting down tftp server")
		s.Shutdown()
	}()
	if err := Serve(ctx, conn, s); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		r.Logger.Error(err, "listen and serve http")
		return err
	}

	return nil
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
		h.Log.Error(err, "opening root directory failed", "rootDirectory", h.RootDirectory)
		return fmt.Errorf("opening root directory %s: %w", h.RootDirectory, err)
	}
	defer root.Close()

	parts := strings.Split(fullfilepath, "/")

	filename := parts[len(parts)-1]
	filedir := strings.Join(parts[:len(parts)-1], "/")
	prefix := parts[0]

	hasMac := false
	if _, err := net.ParseMAC(prefix); err == nil {
		hasMac = true
	}
	hasSerial := regexp.MustCompile(`^\d{2}[a-z]\d{5}$`).MatchString(prefix)

	if hasMac {
		rootpath := filename
		if len(parts) > 2 {
			rootpath = strings.Join(parts[1:], "/")
		}

		childExists := false
		if !root.Exists(filedir) {
			h.Log.Info("creating directories for %s", rootpath)
			// If the mac address directory does not exist, create it.
			err := root.MkdirAll(filedir, 0755)
			if err != nil {
				h.Log.Error(err, "creating directory failed", "directory", filedir)
				return fmt.Errorf("creating %s: %w", filedir, err)
			}
		} else {
			childExists = root.Exists(fullfilepath)
		}

		if !childExists {
			rootExists := root.Exists(rootpath)

			if rootExists {
				// If the file exists in the new path, but not in the old path, use the new path.
				// This is to support the old path for backwards compatibility.
				newF, err := root.Create(fullfilepath)
				if err != nil {
					h.Log.Error(err, "creating file failed", "filename", filename)
					return fmt.Errorf("creating %s: %w", filename, err)
				}
				defer newF.Close()
				oldF, err := root.Open(rootpath)
				if err != nil {
					h.Log.Error(err, "opening file failed", "filename", rootpath)
					return fmt.Errorf("opening %s: %w", rootpath, err)
				}
				defer oldF.Close()
				_, err = io.Copy(newF, oldF)
				if err != nil {
					h.Log.Error(err, "copying file failed", "filename", rootpath)
					return fmt.Errorf("copying %s to %s: %w", rootpath, filename, err)
				}
			} else if content, ok := uboot.Files[rootpath]; ok {
				if err := h.createFile(root, fullfilepath, content); err != nil {
					return err
				}
			}
		}
	}

	var parsedfilepath string
	if hasSerial {
		parsedfilepath = strings.Join(parts[:], "/")
	} else {
		parsedfilepath = strings.Join(parts, "/")
	}

	if _, err := root.Stat(fullfilepath); err == nil {
		// file exists
		file, err := root.Open(fullfilepath)
		if err != nil {
			errMsg := fmt.Sprintf("opening %s: %s", fullfilepath, err.Error())
			h.Log.Error(err, "file open failed")
			return errors.New(errMsg)
		}
		n, err := rf.ReadFrom(file)
		if err != nil {
			errMsg := fmt.Sprintf("reading %s: %s", fullfilepath, err.Error())
			h.Log.Error(err, "file read failed")
			return errors.New(errMsg)
		}
		h.Log.Info("bytes sent", n)
		return nil

	} else if content, ok := uboot.Files[parsedfilepath]; ok {
		ct := bytes.NewReader(content)
		b, err := rf.ReadFrom(ct)
		if err != nil {
			h.Log.Error(err, "file serve failed", "fullfilepath", fullfilepath, "b", b, "contentSize", len(content))
			return err
		}
		h.Log.Info("file served", "bytesSent", b, "contentSize", len(content))
	} else {
		errMsg := fmt.Sprintf("error checking if file exists: %s", fullfilepath)
		h.Log.Error(err, errMsg)
		return errors.New(errMsg)
	}

	// content, ok := binary.Files[filepath.Base(shortfile)]
	// if !ok {
	// 	err := fmt.Errorf("file [%v] unknown: %w", filepath.Base(shortfile), os.ErrNotExist)
	// 	h.Log.Error(err, "file unknown")
	// 	span.SetStatus(codes.Error, err.Error())
	// 	return err
	// }

	// content, err = binary.Patch(content, t.Patch)
	// if err != nil {
	// 	h.Log.Error(err, "failed to patch binary")
	// 	span.SetStatus(codes.Error, err.Error())
	// 	return err
	// }

	// ct := bytes.NewReader(content)
	// b, err := rf.ReadFrom(ct)
	// if err != nil {
	// 	h.Log.Error(err, "file serve failed", "b", b, "contentSize", len(content))
	// 	span.SetStatus(codes.Error, err.Error())

	// 	return err
	// }
	// h.Log.Info("file served", "bytesSent", b, "contentSize", len(content))
	// span.SetStatus(codes.Ok, filename)

	return nil
}

func (h *Handler) createFile(root *Root, filename string, content []byte) error {
	// If the file does not exist in the new path, but exists in the uboot.Files map, use the map.
	newF, err := root.Create(filename)
	if err != nil {
		h.Log.Error(err, "creating file failed", "filename", filename)
		return fmt.Errorf("creating %s: %w", filename, err)
	}
	defer newF.Close()
	_, err = newF.Write(content)
	if err != nil {
		h.Log.Error(err, "writing file failed", "filename", filename)
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	return nil
}

func (h *Handler) HandleIpxeRead(filename string, rf io.ReaderFrom, content []byte) error {
	content, err := binary.Patch(content, []byte(h.Patch))
	if err != nil {
		h.Log.Error(err, "failed to patch binary")
		return err
	}

	ct := bytes.NewReader(content)
	b, err := rf.ReadFrom(ct)
	if err != nil {
		h.Log.Error(err, "file serve failed", "b", b, "contentSize", len(content))
		return err
	}
	h.Log.Info("file served", "bytesSent", b, "contentSize", len(content))

	return nil
}

// HandleWrite handles TFTP PUT requests. It will always return an error. This library does not support PUT.
func (h *Handler) HandleWrite(filename string, wt io.WriterTo) error {
	root, err := os.OpenRoot(h.RootDirectory)
	if err != nil {
		h.Log.Error(err, "opening root directory failed", "rootDirectory", h.RootDirectory)
		return fmt.Errorf("opening root directory %s: %w", h.RootDirectory, err)
	}
	defer root.Close()

	file, err := root.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		h.Log.Error(err, "opening file failed", "filename", filename)
		return nil
	}
	n, err := wt.WriteTo(file)
	if err != nil {
		h.Log.Error(err, "writing file failed", "filename", filename)
		return nil
	}
	h.Log.Info("bytes received", n)
	return nil

	// err := fmt.Errorf("access_violation: %w", os.ErrPermission)
	// client := net.UDPAddr{}
	// if rpi, ok := wt.(tftp.OutgoingTransfer); ok {
	// 	client = rpi.RemoteAddr()
	// }
	// t.Log.Error(err, "client", client, "event", "put", "filename", filename)

	// return err
}
