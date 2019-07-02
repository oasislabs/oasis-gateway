# Developer-gateway

[![codecov](https://codecov.io/gh/oasislabs/developer-gateway/branch/master/graph/badge.svg?token=3iCQK27Rpu)](https://codecov.io/gh/oasislabs/developer-gateway)

The developer-gateway is a component that works along with the Oasis infrastructure to to provide a similar interface to other cloud services. It abstracts the semantics of account wallets typical from blockchains, and provides developers an interface to build applications against a blockchain but with a common user experience, as found in most internet web services.

## Code Organization
The code is organized in the following packages:
 - `[api](api)` APIs exposed by the developer-gateway, the endpoints and the requests and responses for those APIs
 - `[auth](auth)` policies and generic implementations that can be set up from the configuration
 - `[backend](backend)` manages a client implementation and an mqueue implementation to satisfy client requests and provide the responses to clients
 - [callback](callback) callback system implementation
 - [cmd](cmd) contains all the code for the generated binaries from this repository
 - [concurrent](concurrent) contains utilities for common patters to work with concurrent code
 - [config](config) defines how configuration parameters are handled
 - [ekiden](ekiden) implementation of the protocol to talk to ekiden
 - [errors](errors) definition of the error type used in the developer-gateway
 - [eth](eth) abstraction on top of go-ethereum for 
 - [gateway](gateway) creates and binds all services together to generate the developer-gateway request router
 - [log](log) logging package
 - [mqueue](mqueue) message queues implementations used in the backend to keep client messages
 - [noise](noise) noise protocol abstraction to be used for ekiden 
 - [rpc](rpc) abstraction of request routers to handle client requests
 - [rw](rw) io utilities
 - [stats](stats) package to gather and expose simple statistics
 - [tests](tests) component tests
 - [tx](tx) abstraction to execute multiple transactions concurrently

## Build
The command to build all the code, run the unit tests and generate all the repository binaries is `$ make`.

The binaries generated are 
 - `developer-gateway`, which is the binary to run the gateway itself, 
 - `ekiden-client`, which uses the same implementation the `developer-gateway` uses to talk to `ekiden`
 - `eth-client`, which uses the same implementation the `developer-gateway` uses to talk to a web3 gateway.

In order to quickly run the developer-gateway, there's a simple configuration file that can be used for local testing:

```
./developer-gateway --config.path cmd/gateway/config/testing.toml
```

## Testing
The tests are organized in unit tests and component tests. 
 - Unit tests are the tests in each module that test a single unit of code, mocking all the other dependencies the code might have `$ make test`.
 - Component tests test all the code in the developer-gateway component mocking the backend client implementation. This allows to test all the code in the gateway itself independently from the backend used. This tests also run with the different `mqueue` implementations provided `$ make test-component`. Look at the Makefile `test-component-*` to see the different instances of component tests that can be executed
 
## Docs
There is more documentation provided in the [docs](docs) folder

