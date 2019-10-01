package apitest

// Route defines how a request will be multiplexed. A request
// endpoint should be uniquely identified by a route
type Route struct {
	// Method that will be executed on the resource
	Method string

	// Path that indicates the resource or action to
	// be executed
	Path string
}

// Request identifies a request sent by a client and
// handled by a server
type Request struct {
	// Route that uniquely identifies the handler that should
	// handle the request
	Route Route

	// Body of the request
	Body []byte

	// Headers defines metadata to specify different options on how
	// the request may be handled
	Headers map[string]string
}

// Response to a request
type Response struct {
	// Body is the content of the response
	Body []byte

	// Code may be used to identify whether the request
	// was handled successfully or not
	Code int
}
