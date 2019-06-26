package gatewaytest

import "reflect"

// Provider implements a simple service provider
// based on the type of the service. Only one
// instance of a specific type may be added
type Provider struct {
	services []interface{}
}

// Add adds a new type to the provider. Returns true
// on success, false if an instance of the same type has
// already been added
func (p *Provider) Add(v interface{}) bool {
	for _, s := range p.services {
		if reflect.TypeOf(s) == reflect.TypeOf(v) {
			return false
		}
	}

	p.services = append(p.services, v)
	return true
}

// MustAdd adds a new type to the provider. If there is
// already an instance of that type the method will panic
func (p *Provider) MustAdd(v interface{}) {
	if ok := p.Add(v); !ok {
		panic("attempt to add an instance of the same type")
	}
}

// Get retrieves the first instance of that type, or if v
// is an interface, that implements that type.
func (p *Provider) Get(v reflect.Type) (interface{}, bool) {
	for _, s := range p.services {
		t := reflect.TypeOf(s)
		if t == v || (v.Kind() == reflect.Interface && t.Implements(v)) {
			return s, true
		}
	}

	return nil, false
}

// MustGet as get retrieves the first instance of a type, but
// panics if not found
func (p *Provider) MustGet(v reflect.Type) interface{} {
	if t, ok := p.Get(v); ok {
		return t
	}
	panic("instance not found for requested type")
}
