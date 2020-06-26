package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oasislabs/oasis-gateway/errors"
	"github.com/oasislabs/oasis-gateway/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/viper"
)

// An instrumentation service is a background service used to collect
// metrics for a running service.
type instrumentationService interface {
	// StartInstrumentation starts instrumentation tracking for the calling service.
	StartInstrumentation()

	// StopInstrumentation stops instrumentation tracking for the calling service.
	StopInstrumentation()
}

// New constructs a new instrumentation service.
func New(config *MetricsConfig, pkg string, logger log.Logger) (instrumentationService, error) {
	mode := strings.ToLower(viper.GetString(cfgMetricsMode))

	switch mode {
	case metricsModeNone:
		return newStubService()
	case metricsModePull:
		return newPullService(config, pkg, logger)
	case metricsModePush:
		return newPushService(config, pkg, logger)
	default:
		return nil, fmt.Errorf("metrics: unsupported mode: '%v'", mode)
	}
}

// A stub service is a stub instrumentation service.
type stubService struct{}

func newStubService() (instrumentationService, error) {
	return &stubService{}, nil
}

// StartInstrumentation implements the instrumentation service interface for stubService.
func (s *stubService) StartInstrumentation() {}

// StopInstrumentation implements the instrumentation service interface for stubService.
func (s *stubService) StopInstrumentation() {}

// A pull service is a service which exposes metrics that Prometheus can pull.
type pullService struct {
	// Push service context.
	ctx context.Context

	// The HTTP server which hosts the Prometheus metrics endpoint.
	server *http.Server

	// A channel on which to send pull service error messages.
	errCh chan error

	// A logger, for logging.
	logger log.Logger
}

func newPullService(config *MetricsConfig, pkg string, logger log.Logger) (instrumentationService, error) {
	return &pullService{
		ctx: context.Background(),
		server: &http.Server{
			Addr:           fmt.Sprintf("%s:%s", config.PullAddr, config.PullPort),
			Handler:        promhttp.Handler(),
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		logger: logger,
	}, nil
}

// StartInstrumentation implements the instrumentation service interface for pullService.
func (s *pullService) StartInstrumentation() {
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			s.errCh <- err
		}
	}()
}

// StopInstrumentation implements the instrumentation service interface for pullService.
func (s *pullService) StopInstrumentation() {
	if s.server != nil {
		s.server.Shutdown(s.ctx)
		s.server = nil
	}
}

// A push service is used to push metrics to Prometheus.
type pushService struct {
	// Push service context.
	ctx context.Context

	// The pusher which pushes updates to Prometheus.
	pusher *push.Pusher

	// The frequency with which to push updates to Prometheus.
	interval time.Duration

	// A logger, for logging.
	logger log.Logger
}

func newPushService(config *MetricsConfig, pkg string, logger log.Logger) (instrumentationService, error) {
	for _, v := range []string{
		config.PushAddr,
		config.PushJobName,
		config.PushInstanceLabel,
	} {
		if viper.GetString(v) == "" {
			return nil, fmt.Errorf("metrics: %s required for push mode", v)
		}
	}

	pusher := push.New(config.PushAddr, config.PushJobName).
		Grouping("instance", config.PushInstanceLabel).
		Gatherer(prometheus.DefaultGatherer)

	return &pushService{
		ctx:      context.Background(),
		pusher:   pusher,
		interval: config.PushInterval,
		logger:   logger,
	}, nil
}

// StartInstrumentation implements the instrumentation service interface for pushService.
func (s *pushService) StartInstrumentation() {
	go s.startWorker()
}

// StopInstrumentation implements the instrumentation service interface for pushService.
func (s *pushService) StopInstrumentation() {}

func (s *pushService) startWorker() {
	t := time.NewTicker(s.interval)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return

		case <-t.C:
			if err := s.pusher.Push(); err != nil {
				err := errors.New(errors.ErrPrometheusPushError, err)
				s.logger.Error(s.ctx, "metrics: unable to push to prometheus", err)
			}
		}
	}
}
