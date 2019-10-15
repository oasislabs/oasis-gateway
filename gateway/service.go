package gateway

import (
	"fmt"

	"github.com/oasislabs/oasis-gateway/rpc"
	"github.com/oasislabs/oasis-gateway/stats"
)

// Service is the interface that should be implemented
// by all services used by the gateway
type Service interface {
	stats.Collector

	// Name returns a human readable identifier for a service.
	// Each service should have a unique name
	Name() string
}

// Services contains all the services that are exposed
// externally on the gateway and makes them accessible
type Services map[string]Service

// NewServices returns a new instance of services
func NewServices() Services {
	return Services(make(map[string]Service))
}

// Add adds the service by name to the collection of
// services
func (s Services) Add(service Service) {
	if _, ok := s[service.Name()]; ok {
		panic(fmt.Sprintf("Services already contains service %s", service.Name()))
	}

	s[service.Name()] = service
}

// Get returns the service referred by that name if found
func (s Services) Get(name string) (Service, bool) {
	service, ok := s[name]
	return service, ok
}

// MustGet returns service referred by that name or
// panics if not found
func (s Services) MustGet(name string) Service {
	service, ok := s[name]
	if !ok {
		panic(fmt.Sprintf("Services does not contain service %s", name))
	}

	return service
}

// Contains returns true if there is a service with
// that name
func (s Services) Contains(name string) bool {
	_, ok := s[name]
	return ok
}

// Stats returns the stats of all the services
func (s Services) Stats() stats.Metrics {
	group := make(stats.Metrics)

	for _, service := range s {
		s := service.Stats()
		if s != nil {
			group[service.Name()] = s
		}
	}

	return group
}

// HttpRouterService is a wrapper around *rpc.HttpRouter
// so that it can act as a Service
type HttpRouterService struct {
	name   string
	router *rpc.HttpRouter
}

// Name is the implementation of Service.Name
// for HttpRouterService
func (s HttpRouterService) Name() string {
	return s.name
}

// Stats is the implementation of Service.Stats
// for HttpRouterService
func (s HttpRouterService) Stats() stats.Metrics {
	return s.router.Stats()
}
