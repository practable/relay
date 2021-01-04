# booking

To generate the server code, go to root of the repo then

```
swagger generate server -t pkg/booking -f ./api/openapi-spec/booking.yml --exclude-main -A booking
```

If making large changes, then there will be stale files left, so first
```
cd pkg/booking
rm -rf models
rm -rf restapi
```
