package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/appkins-org/go-redfish-uefi/api/redfish"
	"github.com/appkins-org/go-redfish-uefi/internal/config"
	itftp "github.com/appkins-org/go-redfish-uefi/internal/tftp"
	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"golang.org/x/sync/errgroup"

	"github.com/appkins-org/go-redfish-uefi/internal/dhcp/handler/proxy"
	"github.com/appkins-org/go-redfish-uefi/internal/dhcp/server"
	dhcpServer "github.com/appkins-org/go-redfish-uefi/internal/dhcp/server"
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

	dh, err := dhcpHandler(cfg, ctx, log)
	if err != nil {
		log.Error(err, "failed to create dhcp listener")
		panic(fmt.Errorf("failed to create dhcp listener: %w", err))
	}
	log.Info("starting dhcp server", "bind_addr", cfg.Dhcp.Address)
	g.Go(func() error {

		dhcpIp, err := netip.ParseAddrPort(fmt.Sprintf("%s:%d", cfg.Dhcp.Address, cfg.Dhcp.Port))
		if err != nil {
			return fmt.Errorf("invalid bind address: %w", err)
		}

		bindAddr, err := netip.ParseAddrPort(dhcpIp.String())
		if err != nil {
			panic(fmt.Errorf("invalid tftp address for DHCP server: %w", err))
		}
		conn, err := server4.NewIPv4UDPConn(cfg.Dhcp.Interface, net.UDPAddrFromAddrPort(bindAddr))
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		ds := &dhcpServer.DHCP{Logger: log, Conn: conn, Handlers: []dhcpServer.Handler{dh}}

		go func() {
			<-ctx.Done()
			conn.Close()
			ds.Conn.Close()
			ds.Close()
		}()
		return ds.Serve(ctx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error(err, "failed running all services")
		panic(err)
	}
	log.Info("shutting down")

}

func dhcpHandler(c *config.Config, ctx context.Context, log logr.Logger) (server.Handler, error) {
	// 1. create the handler
	// 2. create the backend
	// 3. add the backend to the handler
	pktIP, err := netip.ParseAddr(c.Dhcp.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid bind address: %w", err)
	}
	tftpIP, err := netip.ParseAddrPort(fmt.Sprintf("%s:%d", c.Dhcp.TftpAddress, c.Dhcp.TftpPort))
	if err != nil {
		return nil, fmt.Errorf("invalid tftp address for DHCP server: %w", err)
	}
	httpBinaryURL := &url.URL{
		Scheme: c.Dhcp.IpxeBinaryUrl.Scheme,
		Host:   fmt.Sprintf("%s:%d", c.Dhcp.IpxeBinaryUrl.Address, c.Dhcp.IpxeBinaryUrl.Port),
		Path:   c.Dhcp.IpxeBinaryUrl.Path,
	}
	if _, err := url.Parse(httpBinaryURL.String()); err != nil {
		return nil, fmt.Errorf("invalid http ipxe binary url: %w", err)
	}

	var httpScriptURL *url.URL
	if c.Dhcp.IpxeHttpScriptURL != "" {
		httpScriptURL, err = url.Parse(c.Dhcp.IpxeHttpScriptURL)
		if err != nil {
			return nil, fmt.Errorf("invalid http ipxe script url: %w", err)
		}
	} else {
		httpScriptURL = &url.URL{
			Scheme: c.Dhcp.IpxeBinaryUrl.Scheme,
			Host: func() string {
				switch c.Dhcp.IpxeBinaryUrl.Scheme {
				case "http":
					if c.Dhcp.IpxeBinaryUrl.Port == 80 {
						return c.Dhcp.IpxeBinaryUrl.Address
					}
				case "https":
					if c.Dhcp.IpxeBinaryUrl.Port == 443 {
						return c.Dhcp.IpxeBinaryUrl.Address
					}
				}
				return fmt.Sprintf("%s:%d", c.Dhcp.IpxeBinaryUrl.Address, c.Dhcp.IpxeBinaryUrl.Port)
			}(),
			Path: c.Dhcp.IpxeBinaryUrl.Path,
		}
	}

	if _, err := url.Parse(httpScriptURL.String()); err != nil {
		return nil, fmt.Errorf("invalid http ipxe script url: %w", err)
	}
	ipxeScript := func(*dhcpv4.DHCPv4) *url.URL {
		return httpScriptURL
	}

	ipxeScript = func(d *dhcpv4.DHCPv4) *url.URL {
		u := *httpScriptURL
		p := path.Base(u.Path)
		u.Path = path.Join(path.Dir(u.Path), d.ClientHWAddr.String(), p)
		return &u
	}

	// backend, err := c.backend(ctx, log)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create backend: %w", err)
	// }

	dh := &proxy.Handler{
		// Backend: backend,
		IPAddr: pktIP,
		Log:    log,
		Netboot: proxy.Netboot{
			IPXEBinServerTFTP: tftpIP,
			IPXEBinServerHTTP: httpBinaryURL,
			IPXEScriptURL:     ipxeScript,
			Enabled:           true,
		},
		OTELEnabled:      true,
		AutoProxyEnabled: true,
	}
	return dh, nil

	return nil, errors.New("invalid dhcp mode")
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
