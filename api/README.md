# API definition

## Execute Service

```
curl -X POST http://localhost:1234/v0/api/service/execute -i \
  -H 'X-INSECURE-AUTH:example' -H 'Content-type:application/json' \
  -d '{"data" : "data", "address": "address"}'
```

## Poll Service

```
curl -X POST http://localhost:1234/v0/api/service/poll -i \
  -H 'X-INSECURE-AUTH:example' -H 'Content-type:application/json' \
  -d '{"offset" : 0}'
```

## Deploy Service

```
curl -X POST localhost:1234/v0/api/service/deploy -i \
  -H 'Content-type:application/json'  -H 'X-INSECURE-AUTH:example' \
  -d '{"data" : " some data "}'
```
