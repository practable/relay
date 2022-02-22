# bc

Package `bc` is a client for `pkg/booking` server

## Swagger

Generate client code with swagger
```
swagger generate client -t pkg/bc -f ./api/openapi-spec/booking.yml -A bc
```

If making large changes, then there will be stale files left, so first
```
cd pkg/bc
rm -rf models
rm -rf restapi
```
