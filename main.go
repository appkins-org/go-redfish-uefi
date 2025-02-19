package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/appkins-org/go-redfish-uefi/api/redfish"
	"github.com/gin-gonic/gin"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi.yaml https://opendev.org/airship/go-redfish/raw/branch/master/spec/openapi.yaml

func main() {
	conf, err := NewConfig()
	if err != nil {
		log.Default().Fatal(err)
		panic(err)
	}

	server := redfish.NewRedfishServer(redfish.RedfishServerConfig{
		Insecure:      true,
		UnifiUser:     conf.Unifi.Username,
		UnifiPass:     conf.Unifi.Password,
		UnifiEndpoint: conf.Unifi.Endpoint,
		UnifiSite:     conf.Unifi.Site,
		UnifiDevice:   conf.Unifi.Device,
	})

	addr := fmt.Sprintf("%s:%d", conf.Address, conf.Port)

	h := gin.Default()

	redfish.RegisterHandlers(h, server)

	s := &http.Server{
		Handler: h,
		Addr:    addr,
	}

	// And we serve HTTP until the world ends.
	log.Fatal(s.ListenAndServe())
}
