package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "sre_performance"

var sharedLabels = []string{"budget_name", "service_name"}

var SLIMesurement = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "sli_measurement",
	Help:      "SLI measurement",
}, sharedLabels)

var SLOTarget = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "slo_target",
	Help:      "SLO targets",
}, sharedLabels)

var SLOAvailable = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "slo_budget_available",
	Help:      "SLO available budget",
}, sharedLabels)

var SLIValid = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "sli_valid_events",
	Help:      "SLI valid values",
}, sharedLabels)

var SLIGood = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "sli_good_events",
	Help:      "SLI good events",
}, sharedLabels)

var SLIBad = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "sli_bad_events",
	Help:      "SLI bad events",
}, sharedLabels)

var CalculationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "calculation_duration_seconds",
	Help:      "duration of calculation",
}, sharedLabels)

var CalculationErrors = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "calculation_errors",
	Help:      "errors in calculations",
}, sharedLabels)

func GetHandler() http.Handler {
	return promhttp.Handler()
}
