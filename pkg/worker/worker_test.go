package worker_test

import (
	"testing"

	"github.com/maksim-paskal/sre-metrics-exporter/pkg/worker"
)

func TestCalculateBad(t *testing.T) {
	t.Parallel()

	tests := map[float64]worker.Calculation{
		1.0: {
			Good:  0,
			Valid: 1,
		},
		0: {
			Good:  10,
			Valid: 9,
		},
	}

	for expected, calculation := range tests {
		calculation.CalculateBad()

		if bad := calculation.GetBad(); bad != expected {
			t.Errorf("Expected %f, got %f", expected, bad)
		}
	}
}

func TestCalculateAvailable(t *testing.T) {
	t.Parallel()

	tests := map[float64]worker.Calculation{
		-3.0: {
			Bad:   4,
			Valid: 1000,
		},
		96.0: {
			Bad:   4,
			Valid: 100000,
		},
	}

	for expected, calculation := range tests {
		if available := calculation.GetAvailable(0.999); available != expected {
			t.Errorf("Expected %f, got %f", expected, available)
		}
	}
}
