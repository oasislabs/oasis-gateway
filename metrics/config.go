package metrics

import (
	"time"

	"github.com/oasislabs/oasis-gateway/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	cfgMetricsMode              = "metrics.mode"
	cfgMetricsPullAddr          = "metrics.pull.addr"
	cfgMetricsPullPort          = "metrics.pull.port"
	cfgMetricsPushAddr          = "metrics.push.addr"
	cfgMetricsPushJobName       = "metrics.push.job_name"
	cfgMetricsPushInstanceLabel = "metrics.push.instance_label"
	cfgMetricsPushInterval      = "metrics.push.interval"

	metricsModeNone = "none"
	metricsModePull = "pull"
	metricsModePush = "push"

	defaultPushInterval = 10 // in seconds
)

type MetricsConfig struct {
	Mode              string
	PullAddr          string
	PullPort          string
	PushAddr          string
	PushJobName       string
	PushInstanceLabel string
	PushInterval      time.Duration
}

func (m *MetricsConfig) Log(fields log.Fields) {
	fields.Add(cfgMetricsMode, m.Mode)
	fields.Add(cfgMetricsPullAddr, m.PullAddr)
	fields.Add(cfgMetricsPullPort, m.PullPort)
	fields.Add(cfgMetricsPushAddr, m.PushAddr)
	fields.Add(cfgMetricsPushJobName, m.PushJobName)
	fields.Add(cfgMetricsPushInstanceLabel, m.PushInstanceLabel)
	fields.Add(cfgMetricsPushInterval, m.PushInterval)
}

func (m *MetricsConfig) Configure(v *viper.Viper) error {
	m.Mode = v.GetString(cfgMetricsMode)
	m.PullAddr = v.GetString(cfgMetricsPullAddr)
	m.PullPort = v.GetString(cfgMetricsPullPort)
	m.PushAddr = v.GetString(cfgMetricsPushAddr)
	m.PushJobName = v.GetString(cfgMetricsPushJobName)
	m.PushInstanceLabel = v.GetString(cfgMetricsPushInstanceLabel)
	m.PushInterval = v.GetDuration(cfgMetricsPushInterval)

	return nil
}

func (m *MetricsConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String(cfgMetricsMode, metricsModeNone, "Prometheus metrics mode. Must be one of none, push, pull.")
	cmd.PersistentFlags().String(cfgMetricsPullAddr, "localhost", "Prometheus metrics address, on which the metrics service will live.")
	cmd.PersistentFlags().String(cfgMetricsPullPort, "7000", "Prometheus metrics port, by which service metrics will be made available.")
	cmd.PersistentFlags().String(cfgMetricsPushAddr, "", "Prometheus push gateway address")
	cmd.PersistentFlags().String(cfgMetricsPushJobName, "", "Prometheus push job name")
	cmd.PersistentFlags().String(cfgMetricsPushInstanceLabel, "", "Prometheus push instance label")
	cmd.PersistentFlags().Duration(cfgMetricsPushInterval, defaultPushInterval*time.Second, "Prometheus push interval")

	return nil
}
