package typescript

import (
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// zeroCounts returns a map holding every metric at zero. An analyzer counts
// every metric, enabled or not: the pipeline drops the ones the
// configuration disables.
func zeroCounts() map[config.MetricID]int {
	counts := make(map[config.MetricID]int, len(config.Metrics()))
	for _, m := range config.Metrics() {
		counts[m] = 0
	}
	return counts
}
