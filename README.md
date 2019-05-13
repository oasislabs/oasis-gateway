# Developer-gateway

[![codecov](https://codecov.io/gh/oasislabs/developer-gateway/branch/master/graph/badge.svg?token=3iCQK27Rpu)](https://codecov.io/gh/oasislabs/developer-gateway)

The developer-gateway is a component that works along with the Oasis infrastructure to to provide a similar interface to other cloud services. It abstracts the semantics of account wallets typical from blockchains, and provides developers an interface to build applications against a blockchain but with a common user experience, as found in most internet web services.

## Build
In order to build the developer-gateway.

```
$ make
```

A binary `gateway` will be generated that is the gateway itself. In order to run the gateway, make sure you set the right `private_key` parameter in the wallet in cmd/config/production.toml (feel free to copy the file and generate another configuration if needed). By default, `--config` points to `cmd/gateway/config/production.toml`

```
./gateway --config cmd/gateway/config/production.toml
```

