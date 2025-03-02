package redfish

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/appkins-org/go-redfish-uefi/internal/dhcp/data"
	"github.com/appkins-org/go-redfish-uefi/internal/dhcp/handler"
	"github.com/appkins-org/go-redfish-uefi/internal/firmware/varstore"
	"github.com/go-logr/logr"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func ptr[T any](v T) *T {
	return &v
}

func (s *PowerState) GetPoeMode() string {
	if s == nil {
		return ""
	}
	switch *s {
	case On:
		return "auto"
	case Off:
		return "off"
	case PoweringOff:
		return "off"
	case PoweringOn:
		return "auto"
	default:
		return "off"
	}
}

type RedfishServerConfig struct {
	Insecure      bool
	UnifiUser     string
	UnifiPass     string
	UnifiEndpoint string
	UnifiSite     string
	UnifiDevice   string
	Logger        logr.Logger
	TftpRoot      string
}

type RedfishSystem struct {
	MacAddress       string `yaml:"mac"`
	IpAddress        string `yaml:"ip"`
	UnifiPort        int    `yaml:"port"`
	SiteID           string `yaml:"site"`
	DeviceMac        string `yaml:"device_mac"`
	PoeMode          string `yaml:"poe_mode"`
	EfiVariableStore *varstore.EfiVariableStore
}

func (r *RedfishSystem) GetPowerState() *PowerState {
	state := Off
	switch r.PoeMode {
	case "auto":
		state = On
	case "off":
		state = Off
	default:
		state = Off
	}
	return &state
}

func redfishError(err error) *RedfishError {
	return &RedfishError{
		Error: RedfishErrorError{
			Message: ptr(err.Error()),
			Code:    ptr("Base.1.0.GeneralError"),
		},
	}
}

type RedfishServer struct {
	Systems map[int]RedfishSystem

	Config *RedfishServerConfig

	client *unifi.Client

	Logger logr.Logger

	backend handler.BackendStore
}

func NewRedfishServer(cfg RedfishServerConfig, backend handler.BackendStore) *RedfishServer {
	client := unifi.Client{}

	if err := client.SetBaseURL(cfg.UnifiEndpoint); err != nil {
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

	if err := client.Login(context.Background(), cfg.UnifiUser, cfg.UnifiPass); err != nil {
		panic(fmt.Sprintf("failed to login: %s", err))
	}

	rfSystems := make(map[int]RedfishSystem)

	server := &RedfishServer{
		Systems: rfSystems,
		client:  &client,
		Config:  &cfg,
		Logger:  cfg.Logger,
		backend: backend,
	}

	server.refreshSystems(context.Background())

	return server
}

func (s *RedfishServer) refreshSystems(ctx context.Context) (err error) {
	device, err := s.client.GetDeviceByMAC(ctx, s.Config.UnifiSite, s.Config.UnifiDevice)
	if err != nil {
		panic(err)
	}

	if device.PortOverrides == nil {
		panic("no port overrides found")
	}

	for _, port := range device.PortOverrides {

		sys, ok := s.Systems[port.PortIDX]
		if !ok {
			sys = RedfishSystem{
				UnifiPort: port.PortIDX,
				DeviceMac: device.MAC,
				SiteID:    device.SiteID,
			}
		}
		sys.PoeMode = port.PoeMode

		s.Systems[port.PortIDX] = sys
	}

	if clients, err := s.client.ListActiveClients(ctx, s.Config.UnifiSite); err != nil {
		panic(err)
	} else {
		for _, c := range clients {

			if c.UplinkMac == s.Config.UnifiDevice {

				sys, ok := s.Systems[c.SwPort]
				if !ok {
					sys = RedfishSystem{
						UnifiPort: c.SwPort,
						DeviceMac: c.UplinkMac,
						SiteID:    c.SiteID,
					}
				}

				sys.MacAddress = c.Mac
				sys.IpAddress = c.IP

				firmware := strings.Join([]string{s.Config.TftpRoot, sys.MacAddress, "RPI_EFI.fd"}, string(os.PathSeparator))

				sys.EfiVariableStore, err = varstore.NewEfiVariableStore(firmware)
				if err != nil {
					s.Logger.Error(err, "failed to create EFI variable store", "firmware", firmware)
				}

				s.Systems[c.SwPort] = sys
			}
		}
	}

	for _, sys := range s.Systems {

		if sys.MacAddress == "" {
			continue
		}

		dhcp := data.DHCP{}

		if mac, err := net.ParseMAC(sys.MacAddress); err == nil {
			dhcp.MACAddress = mac

			if ip, err := netip.ParseAddr(sys.IpAddress); err == nil {
				dhcp.IPAddress = ip
			}

			s.backend.Put(ctx, mac, &dhcp, nil)

		} else {
			s.Logger.Error(err, "failed to parse MAC address", "mac", sys.MacAddress)

			continue
		}
	}

	return
}

func (s *RedfishServer) updateDevicePort(ctx context.Context, portIdx int, poeMode string) (device *unifi.Device, err error) {
	device, err = s.client.GetDeviceByMAC(ctx, s.Config.UnifiSite, s.Config.UnifiDevice)
	if err != nil {
		return
	}
	for i, p := range device.PortOverrides {
		if p.PortIDX == portIdx {
			device.PortOverrides[i].PoeMode = poeMode
			device.PortOverrides[i].StpPortMode = false
		}
	}
	device, err = s.client.UpdateDevice(ctx, s.Config.UnifiSite, device)
	return
}

func (s *RedfishServer) getPortState(ctx context.Context, macAddress string, p int) (deviceId string, port unifi.DevicePortOverrides, err error) {
	dev, err := s.client.GetDeviceByMAC(ctx, "default", macAddress)
	if err != nil {
		err = fmt.Errorf("error getting device by MAC Address %s: %v", macAddress, err)
		return
	}

	deviceId = dev.ID

	iPort := slices.IndexFunc(dev.PortOverrides, func(pd unifi.DevicePortOverrides) bool {
		return pd.PortIDX == p
	})

	if iPort == -1 {
		err = fmt.Errorf("port %d not found on device %s", p, deviceId)
		return
	}

	port = dev.PortOverrides[iPort]

	return
}

// CreateVirtualDisk implements ServerInterface.
func (s *RedfishServer) CreateVirtualDisk(w http.ResponseWriter, r *http.Request, systemId string, storageControllerId string) {

	req := CreateVirtualDiskRequestBody{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("error decoding request: %s", err)))
		return
	}

	panic("unimplemented")
}

// DeleteVirtualdisk implements ServerInterface.
func (s *RedfishServer) DeleteVirtualdisk(w http.ResponseWriter, r *http.Request, systemId string, storageId string) {
	panic("unimplemented")
}

// EjectVirtualMedia implements ServerInterface.
func (s *RedfishServer) EjectVirtualMedia(w http.ResponseWriter, r *http.Request, managerId string, virtualMediaId string) {
	panic("unimplemented")
}

// FirmwareInventory implements ServerInterface.
func (s *RedfishServer) FirmwareInventory(w http.ResponseWriter, r *http.Request) {

	panic("unimplemented")
}

// FirmwareInventoryDownloadImage implements ServerInterface.
func (s *RedfishServer) FirmwareInventoryDownloadImage(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// GetManager implements ServerInterface.
func (s *RedfishServer) GetManager(w http.ResponseWriter, r *http.Request, managerId string) {
	panic("unimplemented")
}

// GetManagerVirtualMedia implements ServerInterface.
func (s *RedfishServer) GetManagerVirtualMedia(w http.ResponseWriter, r *http.Request, managerId string, virtualMediaId string) {
	panic("unimplemented")
}

// GetRoot implements ServerInterface.
func (s *RedfishServer) GetRoot(w http.ResponseWriter, r *http.Request) {

	root := Root{
		OdataId:        ptr("/redfish/v1"),
		OdataType:      ptr("#ServiceRoot.v1_11_0.ServiceRoot"),
		Id:             ptr("RootService"),
		Name:           ptr("Root Service"),
		RedfishVersion: ptr("1.11.0"),
		Systems: &IdRef{
			OdataId: ptr("/redfish/v1/Systems"),
		},
	}

	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(root)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("error encoding response: %s", err)))
	}
}

// GetSoftwareInventory implements ServerInterface.
func (s *RedfishServer) GetSoftwareInventory(w http.ResponseWriter, r *http.Request, softwareId string) {
	panic("unimplemented")
}

// GetSystem implements ServerInterface.
func (s *RedfishServer) GetSystem(w http.ResponseWriter, r *http.Request, systemId string) {

	err := s.refreshSystems(r.Context())
	if err != nil {
		w.WriteHeader(500)
		return
	}

	systemIdInt, err := strconv.ParseInt(systemId, 10, 64)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("error parsing system id: %s", err)))
		return
	}

	sy := s.Systems[int(systemIdInt)]

	resp := ComputerSystem{
		Id:         &systemId,
		PowerState: sy.GetPowerState(),
		Links: &SystemLinks{
			Chassis:   &[]IdRef{{OdataId: ptr("/redfish/v1/Chassis/1")}},
			ManagedBy: &[]IdRef{{OdataId: ptr("/redfish/v1/Managers/1")}},
		},
		Boot: &Boot{
			BootSourceOverrideEnabled: ptr(BootSourceOverrideEnabledContinuous),
			BootSourceOverrideTarget:  ptr(None),
			BootSourceOverrideTargetRedfishAllowableValues: &[]BootSource{
				Pxe,
				Hdd,
				None,
			},
		},
		Actions: &ComputerSystemActions{
			HashComputerSystemReset: &ComputerSystemReset{
				ResetTypeRedfishAllowableValues: &[]ResetType{
					ResetTypeOn,
					ResetTypeForceOn,
					ResetTypeForceOff,
					ResetTypePowerCycle,
				},
				Target: ptr(fmt.Sprintf("/redfish/v1/Systems/%s/Actions/ComputerSystem.Reset", systemId)),
			},
		},
		OdataId:   ptr(fmt.Sprintf("/redfish/v1/Systems/%s", systemId)),
		OdataType: ptr("#ComputerSystem.v1_11_0.ComputerSystem"),
		Name:      ptr(fmt.Sprintf("System %s", systemId)),
		Status: &Status{
			State: ptr(StateEnabled),
		},
		UUID: ptr(sy.MacAddress),
	}

	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("error marshalling response: %s", err)))
		return
	}

	w.WriteHeader(200)
	w.Write(b)
}

// GetTask implements ServerInterface.
func (s *RedfishServer) GetTask(w http.ResponseWriter, r *http.Request, taskId string) {
	panic("unimplemented")
}

// GetTaskList implements ServerInterface.
func (s *RedfishServer) GetTaskList(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// GetVolumes implements ServerInterface.
func (s *RedfishServer) GetVolumes(w http.ResponseWriter, r *http.Request, systemId string, storageControllerId string) {
	panic("unimplemented")
}

// InsertVirtualMedia implements ServerInterface.
func (s *RedfishServer) InsertVirtualMedia(w http.ResponseWriter, r *http.Request, managerId string, virtualMediaId string) {
	panic("unimplemented")
}

// ListManagerVirtualMedia implements ServerInterface.
func (s *RedfishServer) ListManagerVirtualMedia(w http.ResponseWriter, r *http.Request, managerId string) {
	panic("unimplemented")
}

// ListManagers implements ServerInterface.
func (s *RedfishServer) ListManagers(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// ListSystems implements ServerInterface.
func (s *RedfishServer) ListSystems(w http.ResponseWriter, r *http.Request) {

	ids := make([]IdRef, 0)

	for i := range s.Systems {
		odataId := fmt.Sprintf("/redfish/v1/Systems/%d", i)
		ids = append(ids, IdRef{
			OdataId: &odataId,
		})
	}

	systems := Collection{
		Members:           &ids,
		OdataContext:      ptr("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		OdataType:         "#ComputerSystemCollection.ComputerSystemCollection",
		Name:              ptr("Computer System Collection"),
		OdataId:           "/redfish/v1/Systems",
		MembersOdataCount: ptr(len(ids)),
	}

	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(systems)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("error encoding response: %s", err)))
	}
}

// ResetIdrac implements ServerInterface.
func (s *RedfishServer) ResetIdrac(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// ResetSystem implements ServerInterface.
func (s *RedfishServer) ResetSystem(w http.ResponseWriter, r *http.Request, systemId string) {

	req := ResetSystemJSONRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(500)
		return
	}

	systemIdInt, err := strconv.ParseInt(systemId, 10, 64)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	err = s.refreshSystems(r.Context())
	if err != nil {
		w.WriteHeader(500)
		return
	}

	sys, ok := s.Systems[int(systemIdInt)]
	if !ok {
		w.WriteHeader(404)
		w.Write([]byte("system not found"))
		return
	}

	if sys.PoeMode == "off" {
		_, err := s.updateDevicePort(r.Context(), sys.UnifiPort, "auto")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		sys.PoeMode = "auto"
	} else if *req.ResetType == ResetTypePowerCycle {
		_, err := s.client.ExecuteCmd(r.Context(), s.Config.UnifiSite, "devmgr", unifi.Cmd{
			Command: "power-cycle",
			MAC:     sys.DeviceMac,
			PortIDX: ptr(sys.UnifiPort),
		})
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(204)
		return
	} else {
		switch *req.ResetType {
		case ResetTypeOn:
			_, err := s.updateDevicePort(r.Context(), sys.UnifiPort, "auto")
			if err != nil {
				w.WriteHeader(500)
				return
			}
			w.WriteHeader(204)
			return
		case ResetTypeForceOn:
			_, err := s.updateDevicePort(r.Context(), sys.UnifiPort, "auto")
			if err != nil {
				w.WriteHeader(500)
				return
			}
			w.WriteHeader(204)
			return
		case ResetTypeForceOff:
			_, err := s.updateDevicePort(r.Context(), sys.UnifiPort, "off")
			if err != nil {
				w.WriteHeader(500)
				return
			}
			w.WriteHeader(204)
			return
		}
	}
}

// SetSystem implements ServerInterface.
func (s *RedfishServer) SetSystem(w http.ResponseWriter, r *http.Request, systemId string) {

	req := SetSystemJSONRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(500)
		return
	}

	systemIdInt, err := strconv.ParseInt(systemId, 10, 64)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	err = s.refreshSystems(r.Context())
	if err != nil {
		w.WriteHeader(500)
		return
	}

	sys, ok := s.Systems[int(systemIdInt)]
	if !ok {
		w.WriteHeader(404)
		w.Write([]byte("system not found"))
		return
	}

	poeMode := req.PowerState.GetPoeMode()

	if poeMode != "" && poeMode != sys.PoeMode {

		sys.PoeMode = poeMode

		_, err := s.updateDevicePort(r.Context(), sys.UnifiPort, sys.PoeMode)
		if err != nil {
			w.WriteHeader(500)
			return
		}
	}

	s.Systems[int(systemIdInt)] = sys

	w.WriteHeader(204)
}

// UpdateService implements ServerInterface.
func (s *RedfishServer) UpdateService(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// UpdateServiceSimpleUpdate implements ServerInterface.
func (s *RedfishServer) UpdateServiceSimpleUpdate(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}
