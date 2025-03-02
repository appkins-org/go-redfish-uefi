package redfish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/appkins-org/go-redfish-uefi/internal/config"
	"github.com/appkins-org/go-redfish-uefi/internal/dhcp/data"
	"github.com/appkins-org/go-redfish-uefi/internal/dhcp/handler"
	"github.com/appkins-org/go-redfish-uefi/internal/firmware/varstore"
	"github.com/go-logr/logr"
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
	Config *config.Config

	Logger logr.Logger

	backend handler.BackendStore
}

func NewRedfishServer(cfg *config.Config, logger logr.Logger, backend handler.BackendStore) *RedfishServer {
	server := &RedfishServer{
		Config:  cfg,
		Logger:  logger,
		backend: backend,
	}

	server.Logger.Info("starting redfish server", "address", cfg.Address, "port", cfg.Port)

	server.refreshSystems(context.Background())

	return server
}

func (s *RedfishServer) refreshSystems(ctx context.Context) (err error) {

	s.Logger.Info("refreshing systems", "backend", s.backend)

	s.backend.Sync(ctx)

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

	ctx := r.Context()

	err := s.refreshSystems(ctx)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	systemIdAddr, err := net.ParseMAC(systemId)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("error parsing system id: %s", err)))
		return
	}

	_, _, pwr, err := s.backend.GetByMac(ctx, systemIdAddr)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	resp := ComputerSystem{
		Id:         &systemId,
		PowerState: (*PowerState)(&pwr.State),
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
		UUID: ptr(systemIdAddr.String()),
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

	keys, err := s.backend.GetKeys(r.Context())
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("error getting keys: %s", err)))
		return
	}

	for _, m := range keys {
		odataId := fmt.Sprintf("/redfish/v1/Systems/%s", m)
		ids = append(ids, IdRef{
			OdataId: &odataId,
		})
	}

	response := Collection{
		Members:           &ids,
		OdataContext:      ptr("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		OdataType:         "#ComputerSystemCollection.ComputerSystemCollection",
		Name:              ptr("Computer System Collection"),
		OdataId:           "/redfish/v1/Systems",
		MembersOdataCount: ptr(len(ids)),
	}

	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(response); err != nil {
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

	ctx := r.Context()

	req := ResetSystemJSONRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(500)
		s.Logger.Error(err, "error decoding request")
		return
	}

	systemIdAddr, err := net.ParseMAC(systemId)
	if err != nil {
		w.WriteHeader(500)
		s.Logger.Error(err, "error parsing system id")
		return
	}

	_, _, pwr, err := s.backend.GetByMac(ctx, systemIdAddr)
	if err != nil {
		w.WriteHeader(500)
		s.Logger.Error(err, "error getting system by mac")
		return
	}

	if pwr == nil {
		w.WriteHeader(404)
		s.Logger.Error(errors.New("power not found"), "system not found", "system", systemId)
		return
	}

	if *req.ResetType == ResetTypeOn && pwr.State == string(On) {
		w.WriteHeader(204)
		s.Logger.Info("system already on", "system", systemId)
		return
	}

	if pwr.State == "off" {

		err := s.backend.Put(ctx, systemIdAddr, nil, nil, &data.Power{
			State: "off",
		})
		if err != nil {
			s.Logger.Error(err, "error setting power state", "system", systemId)
			w.WriteHeader(500)
			return
		}
	} else if *req.ResetType == ResetTypePowerCycle {
		err := s.backend.PowerCycle(ctx, systemIdAddr)
		if err != nil {
			w.WriteHeader(500)
			s.Logger.Error(err, "error power cycling system", "system", systemId)
			return
		}
		w.WriteHeader(204)
		return
	} else {
		state := "auto"
		switch *req.ResetType {
		case ResetTypeOn:
			state = "auto"
		case ResetTypeForceOn:
			state = "auto"
		case ResetTypeForceOff:
			state = "off"
		}
		err := s.backend.Put(ctx, systemIdAddr, nil, nil, &data.Power{
			State: state,
		})
		if err != nil {
			w.WriteHeader(500)
			s.Logger.Error(err, "error setting power state", "system", systemId)
			return
		}
	}
}

// SetSystem implements ServerInterface.
func (s *RedfishServer) SetSystem(w http.ResponseWriter, r *http.Request, systemId string) {

	ctx := r.Context()

	req := SetSystemJSONRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(500)
		s.Logger.Error(err, "error decoding request")
		return
	}

	systemIdAddr, err := net.ParseMAC(systemId)
	if err != nil {
		w.WriteHeader(500)
		s.Logger.Error(err, "error parsing system id")
		return
	}

	_, _, pwr, err := s.backend.GetByMac(ctx, systemIdAddr)
	if err != nil {
		w.WriteHeader(500)
		s.Logger.Error(err, "error getting system by mac")
		return
	}

	poeMode := req.PowerState.GetPoeMode()

	if poeMode != "" && pwr.Mode != poeMode {

		pwr.Mode = poeMode

		err := s.backend.Put(ctx, systemIdAddr, nil, nil, pwr)
		if err != nil {
			w.WriteHeader(500)
			s.Logger.Error(err, "error setting power state", "system", systemId)
			return
		}
	}

	w.WriteHeader(204)
	s.Logger.Info("system updated", "system", systemId)
}

// UpdateService implements ServerInterface.
func (s *RedfishServer) UpdateService(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// UpdateServiceSimpleUpdate implements ServerInterface.
func (s *RedfishServer) UpdateServiceSimpleUpdate(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}
