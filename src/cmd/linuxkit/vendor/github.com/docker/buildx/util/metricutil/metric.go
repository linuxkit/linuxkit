package metricutil

import (
	"github.com/docker/buildx/version"
	"go.opentelemetry.io/otel/metric"
)

// Meter returns a Meter from the MetricProvider that indicates the measurement
// comes from buildx with the appropriate version.
func Meter(mp metric.MeterProvider) metric.Meter {
	return mp.Meter(version.Package,
		metric.WithInstrumentationVersion(version.Version))
}
