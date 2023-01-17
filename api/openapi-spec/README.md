# api

## Useful commands 

Use from repo root.

### Validate
```
swagger  validate ./api/openapi-spec/access.yml
swagger  validate ./api/openapi-spec/booking.yml
swagger  validate ./api/openapi-spec/shellaccess.yml
```

### delete configuration (essential if updating mime types produced/consumed)

```
rm ./internal/access/restapi/configure_access.go
rm ./internal/booking/restapi/configure_booking.go
rm ./internal/shellaccess/restapi/configure_shellaccess.go
```

### generate servers

```
swagger generate server -t internal/access -f ./api/openapi-spec/access.yml --exclude-main -A access
swagger generate server -t internal/booking -f ./api/openapi-spec/booking.yml --exclude-main -A booking
swagger generate server -t internal/shellaccess -f ./api/openapi-spec/shellaccess.yml --exclude-main -A shellaccess
```

### generate clients

```
swagger generate client -t pkg/bc -f ./api/openapi-spec/booking.yml -A bc
```
