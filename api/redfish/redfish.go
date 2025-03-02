package redfish

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

func (server *RedfishServer) ListenAndServe(ctx context.Context, handlers map[string]func(http.ResponseWriter, *http.Request)) error {

	m := http.NewServeMux()

	options := StdHTTPServerOptions{
		BaseURL:    server.Config.Address,
		BaseRouter: m,
	}

	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	for path, handler := range handlers {
		m.HandleFunc(path, handler)
	}

	s := &http.Server{
		Handler: HandlerWithOptions(server, options),

		Addr: fmt.Sprintf("%s:%d", server.Config.Address, server.Config.Port),
	}

	go func() {
		<-ctx.Done()
		server.Logger.Info("shutting down http server")
		_ = s.Shutdown(ctx)
	}()
	if err := s.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		server.Logger.Error(err, "listen and serve http")
		return err
	}

	return nil
}
