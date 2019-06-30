# Configuration
The developer-gateway allows to be configured using command line parameters,
a configuration file and environment variables. The order of precedence for
these are:
 1. CLI parameters overwrite
 1. Configuration file settings, who in turn overwrite
 1. Environment variables

The best way to see all the options and the description of what each option does
is calling the help option. All the parameters can be configured using either of
the above mechanisms.

```
$ ./developer-gateway --help

Flags:
      --auth.plugin strings                            plugins for request authentication
      --auth.provider strings                          providers for request authentication (default [insecure])
      --backend.provider string                        provider for the mailbox service. Options are ethereum, ekiden. (default "ethereum")
      --bind_private.http_interface string             interface to bind for http (default "127.0.0.1")
      --bind_private.http_max_header_bytes int32       http max header bytes for http (default 10000)
      --bind_private.http_port int32                   port to listen to for http (default 1234)
      --bind_private.http_read_timeout_ms int32        http read timeout for http interface (default 10000)
      --bind_private.http_write_timeout_ms int32       http write timeout for http interface (default 10000)
      --bind_private.https_enabled                     if set the interface will listen with https. If this option is set, then bind_private.tls_certificate_path and bind_private.tls_private_key_path must be set as well
      --bind_private.tls_certificate_path string       path to the tls certificate for https
      --bind_private.tls_private_key_path string       path to the private key for https
      --bind_public.http_interface string              interface to bind for http (default "127.0.0.1")
      --bind_public.http_max_header_bytes int32        http max header bytes for http (default 10000)
      --bind_public.http_port int32                    port to listen to for http (default 1234)
      --bind_public.http_read_timeout_ms int32         http read timeout for http interface (default 10000)
      --bind_public.http_write_timeout_ms int32        http write timeout for http interface (default 10000)
      --bind_public.https_enabled                      if set the interface will listen with https. If this option is set, then bind_public.tls_certificate_path and bind_public.tls_private_key_path must be set as well
      --bind_public.tls_certificate_path string        path to the tls certificate for https
      --bind_public.tls_private_key_path string        path to the private key for https
      --callback.wallet_out_of_funds.body string       http body for the callback.
      --callback.wallet_out_of_funds.enabled           enables the wallet_out_of_funds callback. This callback will be sent by thegateway when the provided wallet has run out of funds to execute a transaction.
      --callback.wallet_out_of_funds.headers strings   http headers for the callback.
      --callback.wallet_out_of_funds.method string     http method on the request for the callback.
      --callback.wallet_out_of_funds.queryurl string   http query url for the callback.
      --callback.wallet_out_of_funds.sync              whether to send the callback synchronously.
      --callback.wallet_out_of_funds.url string        http url for the callback.
      --config.path string                             sets the configuration file
      --eth.url string                                 url for the eth endpoint
      --eth.wallet.private_keys strings                private keys for the wallet
      --logging.level string                           sets the minimum logging level for the logger (default "debug")
      --mailbox.provider string                        provider for the mailbox service. Options are mem, redis-single, redis-cluster. (default "mem")
      --mailbox.redis_cluster.addrs stringArray        array of addresses for bootstrap redis instances in the cluster (default [127.0.0.1:6379])
      --mailbox.redis_single.addr string               redis instance address (default "127.0.0.1:6379")

```

The convention on how to set the parameters is the following; for a CLI command
such as `--eth.wallet.private_keys`, it can be set in a toml configuration file
as 

```
[eth.wallet]
private_keys = key1,key2
```

And can be set as an environment variable as `OASIS_DG_ETH_WALLET_PRIVATE_KEYS`.
All environment variables are prefixed by `OASIS_DG` and then are the uppercase
representation of the CLI command replacing `.` by `_`.
