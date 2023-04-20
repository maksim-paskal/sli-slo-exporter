package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maksim-paskal/sre-metrics-exporter/internal"
	"github.com/maksim-paskal/sre-metrics-exporter/pkg/config"
	"github.com/maksim-paskal/sre-metrics-exporter/pkg/prometheus"
	log "github.com/sirupsen/logrus"
)

var version = flag.Bool("version", false, "print version information")

func main() {
	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion()) //nolint:forbidigo
		os.Exit(0)
	}

	log.SetFormatter(&log.JSONFormatter{})

	log.Infof("Starting SRE Metrics Exporter %s...", config.GetVersion())

	if err := config.Load(); err != nil {
		log.Fatal(err)
	}

	log.Debugf("Loded config:\n%s", config.String())

	logLevel, err := log.ParseLevel(*config.Get().LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(logLevel)

	if err := prometheus.Init(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// listen for termination
	go func() {
		<-sigs
		cancel()
	}()

	log.RegisterExitHandler(func() {
		cancel()
		time.Sleep(*config.Get().GracefulShutdownPeriod)
	})

	if err := internal.Start(ctx); err != nil {
		log.WithError(err).Error()
	}

	<-ctx.Done()

	log.Info("Shutting down...")
	time.Sleep(*config.Get().GracefulShutdownPeriod)
}
