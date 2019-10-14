# Deployment
The developer gateway accepts different parameter
[configurations](configuration.md). This guide provides an overview on the main
modules the oasis-gateway uses to better understand how to configure them
under different scenarios

## Modules

### Auth
Auth is the module used for authentication of user requests and verification of
the policies implemented. All requests issued to the public endpoint of the
oasis-gateway gateway are authenticated. The requests issued to the
endpoints where the client provides data are verified.

```
--auth.plugin strings                            plugins for request authentication
--auth.provider strings                          providers for request authentication (default [insecure])
```

### Public API
The public API exposed provides the main functionality that clients get from
the oasis-gateway. So, it needs to be exposed somehow to the clients that
need to use it.

```
--bind_public.http_interface string             interface to bind for http (default "127.0.0.1")
--bind_public.http_max_header_bytes int32       http max header bytes for http (default 10000)
--bind_public.http_port int32                   port to listen to for http (default 1234)
--bind_public.http_read_timeout_ms int32        http read timeout for http interface (default 10000)
--bind_public.http_write_timeout_ms int32       http write timeout for http interface (default 10000)
--bind_public.https_enabled                     if set the interface will listen with https. If this
                                                 option is set, then bind_public.tls_certificate_path
                                                 and bind_public.tls_private_key_path must be set as well
--bind_public.tls_certificate_path string       path to the tls certificate for https
--bind_public.tls_private_key_path string       path to the private key for https

--bind_public.http_cors.allowed_credentials       whether credentials are allowed when using CORS (default true)
--bind_public.http_cors.allowed_headers strings   allowed headers for CORS
--bind_public.http_cors.allowed_methods strings   allowed methods for CORS
--bind_public.http_cors.allowed_origins strings   allowed origins for CORS (default [*])
--bind_public.http_cors.enabled                   if set to true the public port will do CORS handling
--bind_public.http_cors.exposed_headers strings   exposed headers for CORS
--bind_public.http_cors.max_age int               exposed headers for CORS (default -1)
```

### Private API
The private API exposed provides health checking and operational APIs useful for
operators but that should not be exposed to the outside world.

```
--bind_private.http_interface string             interface to bind for http (default "127.0.0.1")
--bind_private.http_max_header_bytes int32       http max header bytes for http (default 10000)
--bind_private.http_port int32                   port to listen to for http (default 1234)
--bind_private.http_read_timeout_ms int32        http read timeout for http interface (default 10000)
--bind_private.http_write_timeout_ms int32       http write timeout for http interface (default 10000)
--bind_private.https_enabled                     if set the interface will listen with https. If this
                                                 option is set, then bind_private.tls_certificate_path
                                                 and bind_private.tls_private_key_path must be set as well
--bind_private.tls_certificate_path string       path to the tls certificate for https
--bind_private.tls_private_key_path string       path to the private key for https
```


### Callbacks
The oasis-gateway provides a callback system to expose state changes that
may be relevant to the operator. These callbacks should be kept internal to the
system in production deployments. The endpoints for callbacks
and the payload that will be sent are configurable

```
--callback.wallet_out_of_funds.body string       http body for the callback.
--callback.wallet_out_of_funds.enabled           enables the wallet_out_of_funds callback. This  callback will 
                                                 be sent by thegateway when the provided wallet has run out of
                                                 funds to execute a transaction.
--callback.wallet_out_of_funds.headers strings   http headers for the callback.
--callback.wallet_out_of_funds.method string     http method on the request for the callback.
--callback.wallet_out_of_funds.queryurl string   http query url for the callback.
--callback.wallet_out_of_funds.sync              whether to send the callback synchronously.
--callback.wallet_out_of_funds.url string        http url for the callback.
```

### Mailbox
The mailbox module keeps state for the client to poll events. These events may
be the result of an asynchronous request issued by the client or to a
subscription. There are two different implementations of the mailbox module; an
in memory provider in which the oasis-gateway keeps state in memory and it
is not shared amongst oasis-gateway instances. And a redis provider in which
a single redis instance can be used or it can be set up with redis cluster for a
fault tolerant deployment.

The goal is to keep the oasis-gateway as a completely stateless components
in which oasis-gateways can be shutdown and restarted without affecting the
clients. If a oasis-gateway is shutdown or crashes, the consequences for the
client today can be:

 - An asynchronous request never receives an event, in which case the client
   will assumes that the request has timeout and it can issue another one. Even
   in that case, it's possible that the request itself has been executed but the
   oasis-gateway did not have time to write the event to the mailbox. So the
   client could see an inconsistency there.
   
 - An existing subscription would be lost. The client would just notice that no
   new events are added to the subscription. This is something that can be
   addressed in the future. However, for now, a client may assume that if after
   some time no new events have received for a subscription, it can destroy it
   and recreate it.
   

```
--mailbox.provider string                        provider for the mailbox service. Options are mem,
                                                 redis-single, redis-cluster. (default "mem")
--mailbox.redis_cluster.addrs stringArray        array of addresses for bootstrap redis instances
                                                 in the cluster (default [127.0.0.1:6379])
--mailbox.redis_single.addr string               redis instance address (default "127.0.0.1:6379")

```

### Wallet
Wallet management is very important to make sure that nobody has access to the
funds owned by the wallet. For now, the oasis-gateway only supports a
standard wallet implementation, in which a private key is provided that is
used for signing. As any other options, the private key can be passed as an
environment variable, in the configuration file or as a command line argument.


```
--eth.wallet.private_keys strings                private keys for the wallet
```

## Deployments

### Local testing
For local testing configuration in which the only goal is to be able to test
contracts quickly the easiest approach is to just use the `mem`
`mailbox.provider` and set the `eth.wallet.private_keys` in the configuration
file or the command line directly (assuming that the private key wallet is just
a wallet for testing). A simple command to start the oasis-gateway could be

```
 ./oasis-gateway --eth.url wss://gateway.oasiscloud.io \
 --eth.wallet.private_keys $PRIVATE_KEYS --mailbox.provider mem
 --auth.provider insecure
```

Also, the configuration file provided in `cmd/gateway/config/testing.toml`
provides a simple base configuration that can be used with

```
 ./oasis-gateway --config.path cmd/gateway/config/testing.toml
```

### Production
For a production deployment, there are a few things to keep in mind:

### Auth
If your users have a Google account, we provide a Google Oauth implementation
that can be chosen as an authentication provider. Otherwise, you may want to
implement your own authentication mechanism and load it as a plugin. Do not use
authentication mechanisms that do not verify that the users who send requests
are actually your users

### Public API
The public API needs to be exposed to the clients. Standard practices for
exposed endpoints apply:
 - Developer-gateway should not serve users requests directly. A proxy should be
   deployed in front of the oasis-gateway to rate limit requests
   and make sure that only the public API port is exposed to the clients
 - The proxy should also distribute requests amongst multiple oasis-gateways
   in a balanced fashion. In general, a round robin strategy should work. The
   thing to keep into account is the rate of subscriptions, since they may be
   long lived, the usage of resources could vary dramatically between
   oasis-gateways
 - The oasis-gateway restricts resources per user-session to make sure
   that resources  are not abused. However, users could create new sessions
   every time to bypass this. So, keep this into consideration when defining the
   infrastructure for a oasis-gateway deployment
 - If your application has a web frontend, it is important to set the `--bind_public.http_cors.*`
   options to limit the domains from which applications can make requests to the
   server and have control on what requests the oasis-gateway replies to.

### Private API
The private API should not be publicly exposed. This private API should be used
for operational purposes; health checks and data collection for monitoring.

### Mailbox
For a production deployment, a redis cluster deployment with multiple
oasis-gateway is encouraged. In that case, if a oasis-gateway crashes,
the other oasis-gateways can still serve the same traffic that the crashed
oasis-gateway could. There are still the pending issues of how to handle
missing events to asynchronous requests or how to recover subscriptions, which
will be addressed in the future

```
--mailbox.provider redis-cluster
--mailbox.redis_cluster.addrs 127.0.0.1:6379,127.0.0.1:6380,127.0.0.1:6381
```

### Wallet
The wallet should be kept completely secret. The best approach may be to use a
HSM device to sign transactions and never expose the private key, but this is
not an implemented feature. For now, the best approach may be to set the wallet
private key through the environment variable `OASIS_DG_ETH_WALLET_PRIVATE_KEYS`
execute the oasis-gateway and unset that variable. 

A mechanism to set the value for the wallet could be that as part of the
deployment process an encrypted file with the private key is decrypted and
loaded to the environment. The oasis-gateway is started and then the key is
unset. 
