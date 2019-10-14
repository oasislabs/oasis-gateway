# Policy API
In the developer-gateway there's an API defined to customize the definition and
management of policies. Service providers that depend on the developer-gateway
to provide a service may have their own users and their own authentication and
authorization mechanisms that want to also implement in the developer-gateway
itself. The Policy API should be the standard mechanism to apply any type of
restriction to a user request. Either because the user does not exist, or it
does not have permissions to execute a specific API, these are restrictions that
should be implemmented at this level. This is a guide on how to implement those
policies.

## Policy Definition
A policy is defined by the interface in  `auth.core.Auth`. All policies executed
against a request in the developer-gateway need to implement that interface. The
developer-gateway is in charge of executing all the policies and running the
verifications to ensure the validity and legitimacy of the request. An important
point to keep in mind is that the policies that implement `auth.core.Auth`
should be kept stateless or at least immutable. The same instance is used to
verify multiple requests concurrently, and if state were to be stored, it could
lead to race conditions.

```go
// AuthRequest is the set of data that is relevant when implementing
// policies. It contains the data against which to make any verification
// and decide whether to allow the request to proceed
type AuthRequest struct {
    // API is a human readable identifier for an API. It is unique
    // for each API provided by the developer-gateway
	API     string
    
    // Address is the address of the contract that is targeted by the
    // request, if any
	Address string
    
    // AAD is the extracted AAD from the data that can be accessed in
    // Auth.Verify to ensure that it has the expected value
	AAD     string
    
    // Data is the data field sent by the client in the *http.Request
    // that can be used to execute any verification
	Data    string
}

// Auth defines an interface for the policies defined in the developer-gateway
// and the policies that can be provided as external plugins. It allows a custom
// definition of how to authenticate the issuer of a request and how to authorize
// the action that the user intends with the request
type Auth interface {
    // Name is a human readable string that uniquely identifies
    // the policy amongst all the services provided by the developer-gateway
	Name() string
    
    // Stats returns a set of metrics to be presented to the client
    // when calling the health API
	Stats() stats.Metrics

    // Authenticate the user from the data in the http request. This method
    // should be used by implementors to retrieve all relevant information
    // from a request, and return the metadata that will be relevant for
    // verification as the string in the return value. In case of error
    // either because of missing information, or other, this method
    // should return an error, and the client will not be authorized to
    // proceed with the request, receiving an authorization error.
    //
    // In practice, the provided string value may be an extracted
    // AAD that can be verified later on the call to Verify
	Authenticate(req *http.Request) (string, error)

	// Verify that a specific payload complies with the expected
    // format and authentication data has the expected values.  This
    // method is only called after a successful call to Authenticate.
    // The expected field is the output of the call to Authenticate.
	Verify(req AuthRequest, expected string) error
}
```

## Policy loading and configuration
In terms of how to implement a policy, there are two approaches. In the
developer-gateway repository there are some implementations of Auth.
The `auth.oauth.GoogleOauth` implementation enables Google OAUTH allows
providers to use Google OAUTH for clients. Another implementation
`auth.insecure.InsecureAuth` can be used for testing but should never be enabled
in production. These implementations are in the developer-gateway. If an
approach can be generic enough for multiple parties to be used, it can be added
to the codebase.

For custom policies, the developer-gateway can load Go plugins that contain an
implementation of the `auth.core.Auth` interface. In order to build a plugin,
the [plugin](https://golang.org/pkg/plugin/)'s package has good documentation on
how to compile a module and load it to the developer-gateway.

In order to tell the developer-gateway to load the policies, the option
`--auth.provider` accepts a list of providers, that will be loaded and executed
in order. For instance, `--auth.provider mypolicy1,mypolicy2` would load
`mypolicy1`, would later load `policy2`. Then, for each request, the call
sequence for the policies would be:

 1. `mypolicy1.Authenticate`
 1. `mypolicy2.Authenticate`
 1. `mypolicy1.Verify`
 1. `mypolicy2.Verify`

To see the code of how this is handled, the implementation of handling the
multiple policies can be found in `auth.core.MultiAuth`.

## Example Policies

### JWT authorization
Let's assume that a service provider would like to use a mechanism based on
[JWT](https://jwt.io/) for end user authentication. JWT is based on the idea
that a token issuer generates a token with a specific set of permissions for the
user. A web server can verify the user has indeed permissions for the APIs that
she intends to execute as well as that the token has a valid signature and has
not expired.

A policy for JWT could be implemented with the following code

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/oasislabs/developer-gateway/stats"
)

// JwtHeader is the header in the *http.Request that contains
// the JWT
const JwtHeader = "X-JWT-AUTH"

// JwtVerifier authenticates an *http.Request and verifies
// that the issuer has the right permissions to execute
// the requested API
type JwtVerifier struct {
	successes stats.Counter
	failures  stats.Counter
}

func (v *JwtVerifier) Name() string {
	return "JwtVerifier"
}

func (v *JwtVerifier) Stats() stats.Metrics {
	return stats.Metrics{
		"successes": v.successes.Value(),
		"failures":  v.failures.Value(),
	}
}

// JwtData represents the relevant authentication data
// from the *http.Request that needs to be verified
type JwtData struct {
	// Scope is the scope defined as part of the
	// JWT claims
	Scope string `json:"scope"`

	// Name is the name of the user as part of the
	// JWT claims
	Name string `json:"name"`
}

// Authenticate returns a json encoded JwtData object on success
// with the relevant data for the verification.
func (v *JwtVerifier) Authenticate(req *http.Request) (string, error) {
	value := req.Header.Get(JwtHeader)
	if len(value) == 0 {
		v.failures.Incr()
		return "", fmt.Errorf("missing request header %s", JwtHeader)
	}

	// here we authenticate the token by verifying that the signature
	// is correct
	t, err := jwt.Parse(value, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		v.failures.Incr()
		return "", err
	}

	// collect the relevant data from the token into JwtData
	// and serialize it into JSON to be latter processed
	// by Verify
	p, err := json.Marshal(JwtData{
		Scope: t.Claims.(jwt.MapClaims)["scope"].(string),
		Name:  t.Claims.(jwt.MapClaims)["name"].(string),
	})
	if err != nil {
		return "", err
	}

	return string(p), err
}

// Verify that the data in an encoded JwtData matches the
// verifier expectations and the request can proceed the
// normal flow
func (v *JwtVerifier) Verify(req AuthRequest, encoded string) error {
	var data JwtData
	if err := json.Unmarshal([]byte(encoded), &data); err != nil {
		v.failures.Incr()
		return err
	}

	if data.Name != req.AAD {
		v.failures.Incr()
		return errors.New("request AAD does not match token identity name")
	}

	if data.Scope != req.API {
		v.failures.Incr()
		return errors.New("request API does not match token scope")
	}

	v.successes.Incr()
	return nil
}
```

As we can see in the example, the use `Authenticate` is purely to make sure the
request token has a valid signature and collect relevant information for later
verification. The output of `Authenticate` is later fed to `Verify` to verify
that the expectations on the token data collected are met. 

### External authorization
A common architecture approach is to have an external authentication server that
authenticates all requests coming into the system. In that case, we can
implement a verifier by making requests to the authentication server and allow
the requests to move forward on success. A simple implementation of this
approach could look like:

```go
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/oasislabs/developer-gateway/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ExtAuthHeader is the header in the *http.Request that contains
// the relevant authentication data for the server
const ExtAuthHeader = "X-EXT-AUTH"

type Client interface {
	Do(*http.Request) (*http.Response, error)
}

// ExtAuthVerifier authenticates an *http.Request and verifies
// that the issuer has the right permissions to execute
// the requested API against an external authentication server
type ExtAuthVerifier struct {
	successes stats.Counter
	failures  stats.Counter
	client    Client
}

func NewExtAuthVerifier(client Client) *ExtAuthVerifier {
	return &ExtAuthVerifier{
		client: client,
	}
}

// ExtAuthPayload is the payload sent out to the authentication
// server for request authentication
type ExtAuthPayload struct {
	// RequestData is the data provided by the auth framework to
	// check for the request legitimacy to be executed
	RequestData AuthRequest

	// Token is the data in the header of the *http.Request
	// collected in the call to ExtAuthVerifier.Authenticate
	Token string
}

func (v *ExtAuthVerifier) Name() string {
	return "ExtAuthVerifier"
}

func (v *ExtAuthVerifier) Stats() stats.Metrics {
	return stats.Metrics{
		"successes": v.successes.Value(),
		"failures":  v.failures.Value(),
	}
}

// Authenticate returns the contents of the ExtAuthHeader on success
func (v *ExtAuthVerifier) Authenticate(req *http.Request) (string, error) {
	value := req.Header.Get(ExtAuthHeader)
	if len(value) == 0 {
		v.failures.Incr()
		return "", fmt.Errorf("missing request header %s", ExtAuthHeader)
	}

	return value, nil
}

// Verify makes a request to the external authentication server. This
// method succeeds based on the response
func (v *ExtAuthVerifier) Verify(req AuthRequest, token string) error {
	buffer := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buffer).Encode(ExtAuthPayload{
		RequestData: req,
		Token:       token,
	}); err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", "/authenticate", buffer)
	if err != nil {
		return err
	}

	res, err := v.client.Do(httpReq)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return errors.New("request not authorized")
	}

	return nil
}
```

