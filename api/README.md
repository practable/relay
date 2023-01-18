# api

## Useful commands 

Use from repo root.

### Validate
```
swagger  validate ./api/access.yml
```

### delete configuration (essential if updating mime types produced/consumed)

```
rm ./internal/access/restapi/configure_access.go
```

### generate servers

```
swagger generate server -t internal/access -f ./api/openapi-spec/access.yml --exclude-main -A access
```

### generate clients

```
swagger generate client -t internal/access -f ./api/access.yml -A ac
```
