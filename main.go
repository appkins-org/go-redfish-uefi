package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/appkins-org/go-redfish-uefi/api/redfish"
	"github.com/appkins-org/go-redfish-uefi/internal/config"
	itftp "github.com/appkins-org/go-redfish-uefi/internal/tftp"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -package redfish -o api/redfish/server.gen.go -generate gin-server,models https://opendev.org/airship/go-redfish/raw/branch/master/spec/openapi.yaml
//go:generate go run github.com/rwtodd/Go.Sed/cmd/sed-go -i "s/systemId/ComputerSystemId/g" api/redfish/server.gen.go

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
	}

	log := defaultLogger(cfg.LogLevel)

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()

	g, ctx := errgroup.WithContext(ctx)

	server := redfish.NewRedfishServer(redfish.RedfishServerConfig{
		Insecure:      true,
		UnifiUser:     cfg.Unifi.Username,
		UnifiPass:     cfg.Unifi.Password,
		UnifiEndpoint: cfg.Unifi.Endpoint,
		UnifiSite:     cfg.Unifi.Site,
		UnifiDevice:   cfg.Unifi.Device,
		Logger:        log,
	})

	addr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)

	g.Go(func() error {
		return server.ListenAndServe(ctx, addr)
	})

	ts := &itftp.Server{
		Logger:        log,
		RootDirectory: cfg.Tftp.RootDirectory,
		Patch:         cfg.Tftp.IpxePatch,
	}

	g.Go(func() error {
		return ts.ListenAndServe(ctx, netip.AddrPortFrom(netip.MustParseAddr(cfg.Address), 69))
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error(err, "failed running all services")
		panic(err)
	}
	log.Info("shutting down")

}

// defaultLogger uses the slog logr implementation.
func defaultLogger(level string) logr.Logger {
	// source file and function can be long. This makes the logs less readable.
	// truncate source file and function to last 3 parts for improved readability.
	customAttr := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			ss, ok := a.Value.Any().(*slog.Source)
			if !ok || ss == nil {
				return a
			}
			f := strings.Split(ss.Function, "/")
			if len(f) > 3 {
				ss.Function = filepath.Join(f[len(f)-3:]...)
			}
			p := strings.Split(ss.File, "/")
			if len(p) > 3 {
				ss.File = filepath.Join(p[len(p)-3:]...)
			}

			return a
		}

		return a
	}
	opts := &slog.HandlerOptions{AddSource: true, ReplaceAttr: customAttr}
	switch level {
	case "debug":
		opts.Level = slog.LevelDebug
	default:
		opts.Level = slog.LevelInfo
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, opts))

	return logr.FromSlogHandler(log.Handler())
}
