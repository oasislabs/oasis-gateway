# API definition

## Deploy Service

```
curl -X POST localhost:1234/v0/api/service/deploy -i \
  -H 'Content-type:application/json'  -H 'X-OASIS-INSECURE-AUTH:example' \
  -d '{"data" : " some data "}'
```

## Execute Service

```
curl -X POST http://localhost:1234/v0/api/service/execute -i \
  -H 'X-OASIS-INSECURE-AUTH:example' -H 'Content-type:application/json' \
  -d '{"data" : "data", "address": "address"}'
```

## Poll Service

```
curl -X POST http://localhost:1234/v0/api/service/poll -i \
  -H 'X-OASIS-INSECURE-AUTH:example' -H 'Content-type:application/json' \
  -d '{"offset" : 0}'
```

## Get Code

```
curl -X POST http://localhost:1234/v0/api/service/getCode -i \
  -H 'X-OASIS-INSECURE-AUTH:example' -H 'Content-type:application/json' \
  -d '{"address" : "address"}'
```

## Get Expiry

```
curl -X POST http://localhost:1234/v0/api/service/getExpiry -i \
  -H 'X-OASIS-INSECURE-AUTH:example' -H 'Content-type:application/json' \
  -d '{"address" : "address"}'
```

## Get Public Key

```
curl -X POST http://localhost:1234/v0/api/service/getPublicKey -i \
  -H 'X-OASIS-INSECURE-AUTH:example' -H 'Content-type:application/json' \
  -d '{"address" : "address"}'
```
