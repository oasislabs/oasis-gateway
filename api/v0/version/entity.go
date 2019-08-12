package version

// GetVersionRequest is a request to retrieve the health
// status of the component.
type GetVersionRequest struct{}

// GetVersionResponse is the response to the health request
type GetVersionResponse struct {
	Version int `json:"version"`
}
