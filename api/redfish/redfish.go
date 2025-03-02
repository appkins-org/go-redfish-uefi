package redfish

import (
	"context"
	"errors"
	"net/http"
)

func (server *RedfishServer) ListenAndServe(ctx context.Context, addr string, handlers map[string]func(http.ResponseWriter, *http.Request)) error {
	r := http.NewServeMux()

	options := StdHTTPServerOptions{
		BaseURL:    addr,
		BaseRouter: r,
	}

	for path, handler := range handlers {
		r.HandleFunc(path, handler)
	}

	h := HandlerWithOptions(server, options)

	s := &http.Server{
		Handler: h,

		Addr: addr,
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
