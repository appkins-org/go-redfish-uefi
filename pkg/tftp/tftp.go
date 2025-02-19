package tftp

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pin/tftp/v3"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	ctx           context.Context
	RootDirectory string
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
func (h *Handler) HandleRead(filename string, rf io.ReaderFrom) error {
	root, err := os.OpenRoot(h.RootDirectory)
	if err != nil {
		return fmt.Errorf("opening root directory %s: %w", h.RootDirectory, err)
	}
	defer root.Close()

	if _, err := root.Stat(filename); err == nil {
		// file exists
		file, err := root.Open(filename)
		if err != nil {
			fmt.Printf("opening %s: %w", filename, err)
			return nil
		}
		n, err := rf.ReadFrom(file)
		if err != nil {
			fmt.Printf("reading %s: %w", filename, err)
			return nil
		}
		fmt.Printf("%d bytes sent\n", n)
		return nil

	} else if _, err := net.ParseMAC(path.Dir(filename)); err == nil {
		// If a mac address is provided, use it to find the binary.
		filepaths := strings.Split(filename, "/")
		if len(filepaths) > 1 {
			filepaths = filepaths[1:]
		}

		newFp := filepath.Join(filepaths...)

		if _, err := root.Stat(newFp); err == nil {
			file, err := root.Open(newFp)
			if err != nil {
				fmt.Printf("opening %s: %w", newFp, err)
				return nil
			}
			n, err := rf.ReadFrom(file)
			if err != nil {
				fmt.Printf("reading %s: %w", newFp, err)
				return nil
			}
			fmt.Printf("%d bytes sent\n", n)
			return nil
		} else {
			fmt.Printf("file not found: %v\n", newFp)
			return nil
		}
	} else {
		fmt.Printf("error checking if file exists: %v\n", err)
		return nil
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

// HandleWrite handles TFTP PUT requests. It will always return an error. This library does not support PUT.
func (h *Handler) HandleWrite(filename string, wt io.WriterTo) error {
	root, err := os.OpenRoot(h.RootDirectory)
	if err != nil {
		return fmt.Errorf("opening root directory %s: %w", h.RootDirectory, err)
	}
	defer root.Close()

	file, err := root.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating %s: %v\n", filename, err)
		return nil
	}
	n, err := wt.WriteTo(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", filename, err)
		return nil
	}
	fmt.Printf("%d bytes received\n", n)
	return nil

	// err := fmt.Errorf("access_violation: %w", os.ErrPermission)
	// client := net.UDPAddr{}
	// if rpi, ok := wt.(tftp.OutgoingTransfer); ok {
	// 	client = rpi.RemoteAddr()
	// }
	// t.Log.Error(err, "client", client, "event", "put", "filename", filename)

	// return err
}

// extractTraceparentFromFilename takes a context and filename and checks the filename for
// a traceparent tacked onto the end of it. If there is a match, the traceparent is extracted
// and a new SpanContext is contstructed and added to the context.Context that is returned.
// The filename is shortened to just the original filename so the rest of boots tftp can
// carry on as usual.
func extractTraceparentFromFilename(ctx context.Context, filename string) (context.Context, string, error) {
	// traceparentRe captures 4 items, the original filename, the trace id, span id, and trace flags
	traceparentRe := regexp.MustCompile("^(.*)-[[:xdigit:]]{2}-([[:xdigit:]]{32})-([[:xdigit:]]{16})-([[:xdigit:]]{2})")
	parts := traceparentRe.FindStringSubmatch(filename)
	if len(parts) == 5 {
		traceID, err := trace.TraceIDFromHex(parts[2])
		if err != nil {
			return ctx, filename, fmt.Errorf("parsing OpenTelemetry trace id %q failed: %w", parts[2], err)
		}

		spanID, err := trace.SpanIDFromHex(parts[3])
		if err != nil {
			return ctx, filename, fmt.Errorf("parsing OpenTelemetry span id %q failed: %w", parts[3], err)
		}

		// create a span context with the parent trace id & span id
		spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			Remote:     true,
			TraceFlags: trace.FlagsSampled, // TODO: use the parts[4] value instead
		})

		// inject it into the context.Context and return it along with the original filename
		return trace.ContextWithSpanContext(ctx, spanCtx), parts[1], nil
	}
	// no traceparent found, return everything as it was
	return ctx, filename, nil
}
