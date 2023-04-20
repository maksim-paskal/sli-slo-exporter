package internal

import (
	"context"

	"github.com/maksim-paskal/sre-metrics-exporter/pkg/config"
	"github.com/maksim-paskal/sre-metrics-exporter/pkg/web"
	"github.com/maksim-paskal/sre-metrics-exporter/pkg/worker"
)

func Start(ctx context.Context) error {
	go web.Start(ctx)

	for _, budget := range config.Get().Budgets {
		go worker.Start(ctx, budget, config.Get().ServiceLevelObjectives)
	}

	return nil
}
