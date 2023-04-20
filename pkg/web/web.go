package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/maksim-paskal/sre-metrics-exporter/pkg/config"
	"github.com/maksim-paskal/sre-metrics-exporter/pkg/metrics"
	log "github.com/sirupsen/logrus"
)

const serverReadTimeout = 10 * time.Second

func GetHandler() *http.ServeMux {
	mux := http.NewServeMux()

	// metrics
	mux.Handle("/metrics", metrics.GetHandler())
	mux.HandleFunc("/ready", healthz)
	mux.HandleFunc("/healthz", healthz)

	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "ok")
}

func Start(cxt context.Context) {
	log.Infof("Starting web server %s", *config.Get().WebAddress)

	server := &http.Server{
		Addr:              *config.Get().WebAddress,
		Handler:           http.TimeoutHandler(GetHandler(), serverReadTimeout, "timeout"),
		ReadHeaderTimeout: serverReadTimeout,
	}

	go func() {
		<-cxt.Done()

		ctx, cancel := context.WithTimeout(context.Background(), *config.Get().GracefulShutdownPeriod)
		defer cancel()

		_ = server.Shutdown(ctx) //nolint:contextcheck
	}()

	if err := server.ListenAndServe(); err != nil {
		log.WithError(err).Fatal("error while starting web server")
	}
}
