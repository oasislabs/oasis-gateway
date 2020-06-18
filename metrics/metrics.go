// Package metrics defines mechanisms for instrumentation of Oasis services.
package metrics

import (
	"context"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/oasislabs/oasis-gateway/errorcodes"
	"github.com/oasislabs/oasis-gateway/go/log"
)

const (
	cfgMetricsPushAddr          = "metrics.push.addr"
	cfgMetricsPushJobName       = "metrics.push.job_name"
	cfgMetricsPushInstanceLabel = "metrics.push.instance_label"
	cfgMetricsPushInterval      = "metrics.push.interval"
	cfgLoggingLevel             = "logging.level"

	defaultPushInterval = 10 // in seconds
)

var (
	flagMetricsPushAddr          string
	flagMetricsPushJobName       string
	flagMetricsPushInstanceLabel string
	flagMetricsPushInterval      time.Duration
	flagLoggingLevel             string
)

// RegisterInstrumentation registers instrumentation tracking for the calling service.
func RegisterInstrumentation(metricsCmd *cobra.Command) {
	metricsCmd.PersistentFlags().StringVar(&flagMetricsPushAddr, cfgMetricsPushAddr, "", "Prometheus push gateway address")
	metricsCmd.PersistentFlags().StringVar(&flagMetricsPushJobName, cfgMetricsPushJobName, "", "Prometheus push job name")
	metricsCmd.PersistentFlags().StringVar(&flagMetricsPushInstanceLabel, cfgMetricsPushInstanceLabel, "", "Prometheus push instance label")
	metricsCmd.PersistentFlags().DurationVar(&flagMetricsPushInterval, cfgMetricsPushInterval, defaultPushInterval*time.Second, "Prometheus push interval")
	metricsCmd.PersistentFlags().StringVar(&flagLoggingLevel, cfgLoggingLevel, logrus.WarnLevel.String(), "Threshold of logging messages")

	for _, v := range []string{
		cfgMetricsPushAddr,
		cfgMetricsPushJobName,
		cfgMetricsPushInstanceLabel,
		cfgMetricsPushInterval,
	} {
		viper.BindPFlag(v, metricsCmd.Flags().Lookup(v))
	}
}

// StartInstrumentation starts instrumentation tracking for the calling service.
func StartInstrumentation(ctx context.Context) {
	p := newInstrumentationTracker(flagMetricsPushInstanceLabel, flagLoggingLevel)

	go p.startWorker(ctx, flagMetricsPushInterval)
}

// An instrumentation tracker is used to push metrics to Prometheus.
type instrumentationTracker struct {
	// The pusher which pushes updates to Prometheus.
	pusher *push.Pusher

	// The frequency with which to push updates to Prometheus.
	interval time.Duration

	// A logger, for logging.
	logger log.Logger
}

func newInstrumentationTracker(instanceLabel, loggingLevel string) *instrumentationTracker {
	lvl, err := logrus.ParseLevel(loggingLevel)
	if err != nil {
		lvl = logrus.WarnLevel
	}
	logger := log.NewLogrus(log.LogrusLoggerProperties{
		Level:  lvl,
		Output: os.Stdout,
	}).ForClass(instanceLabel, "Instrumentation")

	pusher := push.New(flagMetricsPushAddr, flagMetricsPushJobName).
		Grouping("instance", flagMetricsPushInstanceLabel).
		Gatherer(prometheus.DefaultGatherer)

	return &instrumentationTracker{
		pusher:   pusher,
		interval: flagMetricsPushInterval,
		logger:   logger,
	}
}

func (i *instrumentationTracker) startWorker(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:
			if err := i.pusher.Push(); err != nil {
				i.logger.Error("unable to push to prometheus", errorcodes.New(errorcodes.ErrPrometheusPushError, err))
			}
		}
	}
}
