// Package file watches a file for changes and updates the in memory DHCP data.
package remote

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"slices"
	"time"

	"github.com/appkins-org/go-redfish-uefi/internal/config"
	"github.com/appkins-org/go-redfish-uefi/internal/dhcp/data"
	"github.com/go-logr/logr"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"go.opentelemetry.io/otel"
)

const tracerName = "github.com/appkins-org/go-redfish-uefi/backend/remote"

var (
	errRecordNotFound = fmt.Errorf("record not found")
)

// Remote represents the backend for watching a file for changes and updating the in memory DHCP data.
type Remote struct {
	// Log is the logger to be used in the File backend.
	Log logr.Logger

	config *config.UnifiConfig

	client *unifi.Client
}

// NewRemote creates a new file watcher.
func NewRemote(l logr.Logger, cfg config.UnifiConfig) (*Remote, error) {

	client := unifi.Client{}

	if err := client.SetBaseURL(cfg.Endpoint); err != nil {
		panic(fmt.Sprintf("failed to set base url: %s", err))
	}

	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.Insecure,
		},
	}

	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	if err := client.SetHTTPClient(httpClient); err != nil {
		panic(fmt.Sprintf("failed to set http client: %s", err))
	}

	if err := client.Login(context.Background(), cfg.Username, cfg.Password); err != nil {
		panic(fmt.Sprintf("failed to login: %s", err))
	}

	return &Remote{
		Log:    l,
		client: &client,
		config: &cfg,
	}, nil
}

// GetByMac is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) GetByMac(ctx context.Context, mac net.HardwareAddr) (*data.DHCP, *data.Netboot, *data.Power, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetByMac")
	defer span.End()

	dhcp := data.DHCP{
		MACAddress: mac,
	}

	power := data.Power{}

	netboot := data.Netboot{}

	if activeClient, err := w.getActiveClientByMac(ctx, mac.String()); err == nil {

		power.Port = activeClient.SwPort

		if ipAddr, err := netip.ParseAddr(activeClient.IP); err == nil {
			dhcp.IPAddress = ipAddr
		}

		dhcp.Hostname = activeClient.Hostname
		if activeClient.VirtualNetworkOverrideID != "" {
			dhcp.VLANID = activeClient.VirtualNetworkOverrideID
		}
		dhcp.LeaseTime = 604800
		dhcp.Arch = "arm64"
		dhcp.Disabled = false

		if network, err := w.client.GetNetwork(ctx, w.config.Site, activeClient.NetworkID); err == nil {

			if _, cidr, err := net.ParseCIDR(network.IPSubnet); err == nil {
				dhcp.SubnetMask = cidr.Mask
			}

			if network.DHCPDGateway != "" {
				if dhcpGateway, err := netip.ParseAddr(network.DHCPDGateway); err == nil {
					dhcp.DefaultGateway = dhcpGateway
				}
			}

			dhcp.NameServers = []net.IP{}

			if network.DHCPDDNS1 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS1))
			}
			if network.DHCPDDNS2 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS2))
			}
			if network.DHCPDDNS3 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS3))
			}
			if network.DHCPDDNS4 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS4))
			}

			dhcp.NTPServers = []net.IP{}

			if network.DHCPDNtp1 != "" {
				dhcp.NTPServers = append(dhcp.NTPServers, net.ParseIP(network.DHCPDNtp1))
			}
			if network.DHCPDNtp2 != "" {
				dhcp.NTPServers = append(dhcp.NTPServers, net.ParseIP(network.DHCPDNtp2))
			}
		} else {
			return nil, nil, nil, err
		}

	} else {
		return nil, nil, nil, err
	}

	if portOverrides, err := w.getPortOverride(ctx, power.Port); err == nil {
		power.State = portOverrides.PoeMode
		power.DeviceId = w.config.Device
		power.SiteId = w.config.Site
		power.Port = portOverrides.PortIDX
	} else {
		return nil, nil, nil, err
	}

	return &dhcp, &netboot, &power, nil
}

func (w *Remote) getActiveClientByMac(ctx context.Context, mac string) (*unifi.ActiveClient, error) {
	clients, err := w.client.ListActiveClients(ctx, w.config.Site)
	if err != nil {
		return nil, err
	}

	i := slices.IndexFunc(clients, func(i unifi.ActiveClient) bool {
		return i.Mac == mac
	})
	if i == -1 {
		return nil, fmt.Errorf("no client found")
	}

	return &clients[i], nil
}

func (w *Remote) getActiveClientByIP(ctx context.Context, ip net.IP) (*unifi.ActiveClient, error) {
	clients, err := w.client.ListActiveClients(ctx, w.config.Site)
	if err != nil {
		return nil, err
	}

	i := slices.IndexFunc(clients, func(i unifi.ActiveClient) bool {
		return i.IP == ip.String()
	})
	if i == -1 {
		return nil, fmt.Errorf("no client found")
	}

	return &clients[i], nil
}

func (w *Remote) getPortOverride(ctx context.Context, port int) (*unifi.DevicePortOverrides, error) {

	device, err := w.client.GetDeviceByMAC(ctx, w.config.Site, w.config.Device)
	if err != nil {
		return nil, err
	}

	idx := slices.IndexFunc(device.PortOverrides, func(i unifi.DevicePortOverrides) bool {
		return i.PortIDX == port
	})
	if idx == -1 {
		return nil, fmt.Errorf("no port 1 found")
	}

	return &device.PortOverrides[idx], nil
}

// GetByIP is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) GetByIP(ctx context.Context, ip net.IP) (*data.DHCP, *data.Netboot, *data.Power, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetByIP")
	defer span.End()

	dhcp := data.DHCP{
		IPAddress: netip.MustParseAddr(ip.String()),
	}

	power := data.Power{}

	netboot := data.Netboot{}

	if activeClient, err := w.getActiveClientByIP(ctx, ip); err == nil {

		power.Port = activeClient.SwPort

		dhcp.IPAddress = netip.MustParseAddr(activeClient.IP)
		dhcp.Hostname = activeClient.Hostname
		if activeClient.VirtualNetworkOverrideID != "" {
			dhcp.VLANID = activeClient.VirtualNetworkOverrideID
		}
		dhcp.LeaseTime = 604800
		dhcp.Arch = "arm64"
		dhcp.Disabled = false

		if network, err := w.client.GetNetwork(ctx, w.config.Site, activeClient.NetworkID); err == nil {

			if _, cidr, err := net.ParseCIDR(network.IPSubnet); err == nil {
				dhcp.SubnetMask = cidr.Mask
			}
			dhcp.DefaultGateway = netip.MustParseAddr(network.DHCPDGateway)

			dhcp.NameServers = []net.IP{}

			if network.DHCPDDNS1 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS1))
			}
			if network.DHCPDDNS2 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS2))
			}
			if network.DHCPDDNS3 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS3))
			}
			if network.DHCPDDNS4 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS4))
			}

			dhcp.NTPServers = []net.IP{}

			if network.DHCPDNtp1 != "" {
				dhcp.NTPServers = append(dhcp.NTPServers, net.ParseIP(network.DHCPDNtp1))
			}
			if network.DHCPDNtp2 != "" {
				dhcp.NTPServers = append(dhcp.NTPServers, net.ParseIP(network.DHCPDNtp2))
			}
		} else {
			return nil, nil, nil, err
		}

	} else {
		return nil, nil, nil, err
	}

	if dhcp.MACAddress.String() == "" {
		return nil, nil, nil, errRecordNotFound
	}

	if portOverrides, err := w.getPortOverride(ctx, power.Port); err == nil {
		power.State = portOverrides.PoeMode
		power.DeviceId = w.config.Device
		power.SiteId = w.config.Site
		power.Port = portOverrides.PortIDX
	} else {
		return nil, nil, nil, err
	}

	return &dhcp, &netboot, &power, nil
}

func (w *Remote) Put(ctx context.Context, mac net.HardwareAddr, d *data.DHCP, n *data.Netboot, p *data.Power) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.Put")
	defer span.End()

	if p != nil {

		device, err := w.client.GetDeviceByMAC(ctx, w.config.Site, w.config.Device)
		if err != nil {
			return err
		}

		i := slices.IndexFunc(device.PortOverrides, func(i unifi.DevicePortOverrides) bool {
			return i.PortIDX == p.Port
		})
		if i == -1 {
			return fmt.Errorf("no port %d found", p.Port)
		}

		if device.PortOverrides[i].PoeMode != p.State {
			device.PortOverrides[i].PoeMode = p.State

			if _, err := w.client.UpdateDevice(ctx, w.config.Site, device); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Remote) GetKeys(ctx context.Context) ([]net.HardwareAddr, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetKeys")
	defer span.End()

	device, err := w.client.GetDeviceByMAC(ctx, w.config.Site, w.config.Device)
	if err != nil {
		return nil, err
	}

	ports := []int{}
	for _, port := range device.PortOverrides {
		ports = append(ports, port.PortIDX)
	}

	clients, err := w.client.ListActiveClients(ctx, w.config.Site)
	if err != nil {
		return nil, err
	}

	var keys []net.HardwareAddr
	for _, client := range clients {
		if !slices.Contains(ports, client.SwPort) {
			continue
		}

		if mac, err := net.ParseMAC(client.Mac); err == nil {
			keys = append(keys, mac)
		}
	}

	return keys, nil
}

// PowerCycle is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) PowerCycle(ctx context.Context, mac net.HardwareAddr) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.PowerCycle")
	defer span.End()

	activeClient, err := w.getActiveClientByMac(ctx, mac.String())
	if err != nil {
		w.Log.Error(err, "failed to get active client by mac")
		return err
	}

	if _, err = w.client.ExecuteCmd(ctx, w.config.Site, "devmgr", unifi.Cmd{
		Command: "power-cycle",
		MAC:     w.config.Device,
		PortIDX: ptr(activeClient.SwPort),
	}); err != nil {

		w.Log.Error(err, "failed to power cycle")
		return err
	}

	return nil
}

func ptr[T any](v T) *T {
	return &v
}

// Start starts watching a file for changes and updates the in memory data (w.data) on changew.
// Start is a blocking method. Use a context cancellation to exit.
func (w *Remote) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.Log.Info("stopping remote")
			return
		}
	}
}

func (w *Remote) Sync(ctx context.Context) error {
	return nil
}
