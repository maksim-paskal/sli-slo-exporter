package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var gitVersion = "dev"

func GetVersion() string {
	return gitVersion
}

type Budget struct {
	Name          string
	WindowSeconds int
}

func (b *Budget) GetPrometheusWindows() string {
	return fmt.Sprintf("[%ds]", b.WindowSeconds)
}

type ServiceLevelObjectiveExpession struct {
	Query string
}

type ServiceLevelObjectiveGoodBadRatio struct {
	Good  string
	Valid string
}

func (goodBad *ServiceLevelObjectiveGoodBadRatio) GetFormatedGoodExpression(budget *Budget) string {
	return fmt.Sprintf("sum(increase(%s%s))", goodBad.Good, budget.GetPrometheusWindows())
}

func (goodBad *ServiceLevelObjectiveGoodBadRatio) GetFormatedValidExpression(budget *Budget) string {
	return fmt.Sprintf("sum(increase(%s%s))", goodBad.Valid, budget.GetPrometheusWindows())
}

func (goodBad *ServiceLevelObjectiveGoodBadRatio) GetRatioExpression(budget *Budget) string {
	return fmt.Sprintf("%s/%s", goodBad.GetFormatedGoodExpression(budget), goodBad.GetFormatedValidExpression(budget))
}

type ServiceLevelObjectiveDistributionCut struct {
	Bucket    string
	Threshold string
}

func (distributionCut *ServiceLevelObjectiveDistributionCut) GetFormatedGoodExpression(budget *Budget) string {
	query := fmt.Sprintf("sum(increase(%s%s))", distributionCut.Bucket, budget.GetPrometheusWindows())

	return strings.ReplaceAll(query, "}", fmt.Sprintf(", le=\"%s\"}", distributionCut.Threshold))
}

func (distributionCut *ServiceLevelObjectiveDistributionCut) GetFormatedValidExpression(budget *Budget) string {
	query := fmt.Sprintf("sum(increase(%s%s))", distributionCut.Bucket, budget.GetPrometheusWindows())

	return strings.ReplaceAll(query, "_bucket{", "_count{")
}

func (distributionCut *ServiceLevelObjectiveDistributionCut) GetRatioExpression(budget *Budget) string {
	return fmt.Sprintf("%s/%s", distributionCut.GetFormatedGoodExpression(budget), distributionCut.GetFormatedValidExpression(budget)) //nolint:lll
}

type ServiceLevelObjective struct {
	Name            string
	Expression      *ServiceLevelObjectiveExpession
	GoodBadRatio    *ServiceLevelObjectiveGoodBadRatio
	DistributionCut *ServiceLevelObjectiveDistributionCut
	Goal            float64
}

func (objectiveExpression *ServiceLevelObjectiveExpession) GetFormatedExpression(budget *Budget) string {
	return strings.ReplaceAll(objectiveExpression.Query, "[window]", budget.GetPrometheusWindows())
}

type Type struct {
	GracefulShutdownPeriod *time.Duration
	ConfigFile             *string
	LogLevel               *string
	WebAddress             *string
	PrometheusURL          *string
	PrometheusUser         *string
	PrometheusPassword     *string
	IntervalSeconds        *int
	Budgets                []*Budget
	ServiceLevelObjectives []*ServiceLevelObjective
}

const (
	defaultIntervalSeconds        = 300
	defaultGracefulShutdownPeriod = 5 * time.Second
)

var config = Type{
	GracefulShutdownPeriod: flag.Duration("graceful-shutdown-period", defaultGracefulShutdownPeriod, "graceful shutdown period"), //nolint:lll
	ConfigFile:             flag.String("config", "config.yaml", "config"),
	LogLevel:               flag.String("log.level", "INFO", "logging level"),
	IntervalSeconds:        flag.Int("interval.seconds", defaultIntervalSeconds, "interval seconds"),
	WebAddress:             flag.String("web.listen-address", ":28180", "Address to listen on for web interface and telemetry."), //nolint:lll
	PrometheusURL:          flag.String("prometheus.url", "", "prometheus url"),
	PrometheusUser:         flag.String("prometheus.user", "", "prometheus basic auth user"),
	PrometheusPassword:     flag.String("prometheus.password", "", "prometheus basic auth password"),
}

func Load() error {
	configByte, err := os.ReadFile(*config.ConfigFile)
	if err != nil {
		return errors.Wrap(err, "error in os.ReadFile")
	}

	err = yaml.Unmarshal(configByte, &config)
	if err != nil {
		return errors.Wrap(err, "error in yaml.Unmarshal")
	}

	return nil
}

func String() string {
	byteConfig, err := yaml.Marshal(config)
	if err != nil {
		return err.Error()
	}

	return string(byteConfig)
}

func Get() *Type {
	return &config
}
