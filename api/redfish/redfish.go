package redfish

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (r *RedfishServer) ListenAndServe(ctx context.Context, addr string) error {
	h := gin.Default()

	RegisterHandlers(h, r)

	s := &http.Server{
		Handler: h,
		Addr:    addr,
	}

	go func() {
		<-ctx.Done()
		r.Logger.Info("shutting down http server")
		_ = s.Shutdown(ctx)
	}()
	if err := s.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		r.Logger.Error(err, "listen and serve http")
		return err
	}

	return nil
}
