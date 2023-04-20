package prometheus

import (
	"context"
	"time"

	"github.com/maksim-paskal/sre-metrics-exporter/pkg/config"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promConfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

var promClient api.Client

func Init() error {
	prometheusConfig := api.Config{
		Address: *config.Get().PrometheusURL,
	}

	if len(*config.Get().PrometheusUser) > 0 {
		prometheusConfig.RoundTripper = promConfig.NewBasicAuthRoundTripper(
			*config.Get().PrometheusUser,
			promConfig.Secret(*config.Get().PrometheusPassword),
			"",
			api.DefaultRoundTripper,
		)
	}

	client, err := api.NewClient(prometheusConfig)
	if err != nil {
		return errors.Wrap(err, "error creating client")
	}

	promClient = client

	return nil
}

func GetMetrics(ctx context.Context, query string) (model.Vector, error) {
	log.Debugf("query: %s", query)

	v1api := prometheusv1.NewAPI(promClient)

	result, warnings, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	if len(warnings) > 0 {
		log.Warn(warnings)
	}

	v, ok := result.(model.Vector)
	if !ok {
		return nil, errors.New("assertion error")
	}

	log.Debugf("result: %v", v)

	return v, nil
}
