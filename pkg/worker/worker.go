package worker

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/maksim-paskal/sre-metrics-exporter/pkg/config"
	"github.com/maksim-paskal/sre-metrics-exporter/pkg/metrics"
	"github.com/maksim-paskal/sre-metrics-exporter/pkg/prometheus"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

const (
	maxCalculatingDuration = 120 * time.Second
	retryCount             = 3
	retryIntervalMs        = 4000
)

func Start(ctx context.Context, budget *config.Budget, objectives []*config.ServiceLevelObjective) { //nolint:cyclop,funlen,lll,nolintlint
	interval := time.Duration(*config.Get().IntervalSeconds) * time.Second

	if *config.Get().IntervalSeconds > budget.WindowSeconds {
		interval = time.Duration(budget.WindowSeconds) * time.Second
	}

	log := log.WithField("budget", budget.Name)

	log.Infof("Starting worker, interval=%s...", interval.String())

	// Setup initial SLO Targets dictionary
	for _, objective := range objectives {
		metrics.SLOTarget.WithLabelValues(budget.Name, objective.Name).Set(objective.Goal)
	}

	for {
		if ctx.Err() != nil {
			return
		}

		// calculate SLI for each objective
		for _, objective := range objectives {
			func(objective *config.ServiceLevelObjective) {
				log := log.WithField("objective", objective.Name)

				calculationStartTime := time.Now()
				sharedLabels := []string{budget.Name, objective.Name}

				ctx, cancel := context.WithTimeout(ctx, maxCalculatingDuration)
				defer cancel()

				var (
					calculation *Calculation
					err         error
				)

				try := 0

				// retryable calculation
				for {
					try++

					calculation, err = calculate(ctx, budget, objective)
					if err == nil {
						break
					}

					log.WithError(err).Errorf("(try=%d) error in calculation", try)
					metrics.CalculationErrors.WithLabelValues(sharedLabels...).Inc()

					if try > retryCount {
						log.Error("try exceeded")

						return
					}

					// wait some time
					time.Sleep(time.Duration(rand.Int31n(retryIntervalMs)) * time.Microsecond) //nolint:gosec
				}

				metrics.SLIMesurement.WithLabelValues(sharedLabels...).Set(calculation.GetResult())
				metrics.SLIGood.WithLabelValues(sharedLabels...).Set(calculation.GetGood())
				metrics.SLIValid.WithLabelValues(sharedLabels...).Set(calculation.GetValid())
				metrics.SLIBad.WithLabelValues(sharedLabels...).Set(calculation.GetBad())

				metrics.SLOAvailable.WithLabelValues(sharedLabels...).Set(calculation.GetAvailable(objective.Goal))

				log.Debug(calculation.String())

				metrics.CalculationDuration.WithLabelValues(sharedLabels...).Observe(time.Since(calculationStartTime).Seconds())
			}(objective)
		}

		time.Sleep(interval)
	}
}

type Calculation struct {
	Good   float64
	Valid  float64
	Bad    float64
	Result float64
}

func (c *Calculation) IsGoal(goal float64) string {
	if c.Result >= goal {
		return "1"
	}

	return "0"
}

func (c *Calculation) SetValid(vector model.Vector) {
	if len(vector) == 0 {
		c.Valid = 0
	} else {
		c.Valid = float64(vector[0].Value)
	}
}

func (c *Calculation) SetGood(vector model.Vector) {
	if len(vector) == 0 {
		c.Good = 0
	} else {
		c.Good = float64(vector[0].Value)
	}
}

func (c *Calculation) SetResult(vector model.Vector) {
	if len(vector) == 0 {
		c.Result = 1
	} else {
		c.Result = float64(vector[0].Value)
	}
}

func (c *Calculation) CalculateBad() {
	c.Bad = c.Valid - c.Good
}

func (c *Calculation) GetGood() float64 {
	if c.Good < 0 {
		return 0
	}

	return c.Good
}

func (c *Calculation) GetValid() float64 {
	if c.Valid < 0 {
		return 0
	}

	return c.Valid
}

func (c *Calculation) GetBad() float64 {
	if c.Bad < 0 {
		return 0
	}

	return c.Bad
}

func (c *Calculation) GetResult() float64 {
	if c.Result > 1 {
		return 1
	}

	return c.Result
}

func (c *Calculation) GetAvailable(goal float64) float64 {
	return c.GetValid() - (c.GetValid() * goal) - c.GetBad()
}

func (c *Calculation) String() string {
	return fmt.Sprintf("Result: %f GoodEvents: %d, BadEvents: %d, ValidEvents: %d",
		c.Result,
		int(c.GetGood()),
		int(c.GetBad()),
		int(c.GetValid()),
	)
}

func calculate(ctx context.Context, budget *config.Budget, objective *config.ServiceLevelObjective) (*Calculation, error) { //nolint:lll,cyclop,funlen
	calculation := Calculation{}

	switch {
	// calculate SLI with prometheus expression
	case objective.Expression != nil:
		objectiveResult, err := prometheus.GetMetrics(ctx, objective.Expression.GetFormatedExpression(budget))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get objectiveResult")
		}

		if len(objectiveResult) == 0 {
			calculation.Result = 0
		} else {
			calculation.Result = float64(objectiveResult[0].Value)
		}
	// calculate SLI with good/bad ratio
	case objective.GoodBadRatio != nil:
		goodResults, err := prometheus.GetMetrics(ctx, objective.GoodBadRatio.GetFormatedGoodExpression(budget))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get goodResults")
		}

		calculation.SetGood(goodResults)

		validResults, err := prometheus.GetMetrics(ctx, objective.GoodBadRatio.GetFormatedValidExpression(budget))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get validResults")
		}

		calculation.SetValid(validResults)
		calculation.CalculateBad()

		if calculation.Bad == 0 {
			// for the case when there are no bad events, we set the SLI to 1
			calculation.Result = 1
		} else {
			// calculate the ratio of good events to valid events
			goodValidRatioResults, err := prometheus.GetMetrics(ctx, objective.GoodBadRatio.GetRatioExpression(budget))
			if err != nil {
				return nil, errors.Wrap(err, "failed to get goodValidRatioResults")
			}

			calculation.SetResult(goodValidRatioResults)
		}
	case objective.DistributionCut != nil:
		validResults, err := prometheus.GetMetrics(ctx, objective.DistributionCut.GetFormatedValidExpression(budget))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get validResults")
		}

		calculation.SetValid(validResults)

		goodResults, err := prometheus.GetMetrics(ctx, objective.DistributionCut.GetFormatedGoodExpression(budget))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get goodResults")
		}

		calculation.SetGood(goodResults)
		calculation.CalculateBad()

		if calculation.Bad == 0 {
			// for the case when there are no bad events, we set the SLI to 1
			calculation.Result = 1
		} else {
			// calculate the ratio of good events to valid events
			distributionCutRatioResults, err := prometheus.GetMetrics(ctx, objective.DistributionCut.GetRatioExpression(budget))
			if err != nil {
				return nil, errors.Wrap(err, "failed to get distributionCutRatioResults")
			}

			calculation.SetResult(distributionCutRatioResults)
		}
	default:
		return nil, errors.New("unknown calculation type")
	}

	return &calculation, nil
}
