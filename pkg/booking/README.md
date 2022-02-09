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


# Note updated manifest with config

This solves #13 

The config file is specified in the manifest :

```
  pvna-activity-00:
    description: pvna-activity-v1.0
    config:
      url: https://assets.practable.io/config/experiments/pvna/pvna00-0.0.json
      <snip>
```

The config key must be included in the url template in the UI description in the manifest

```
  pvna-default-ui-1.0:
   <snip>
    url: https://static.practable.io/ui/pvna-1.0/?config={{config}}&streams={{streams}}&exp={{exp}}
    <snip>
```

The config is passed when the activity is booked

```
<snip> 
"config":{"url":"https://assets.practable.io/config/experiments/pvna/pvna00-0.0.json"}
<snip>
```

Then [bookjs](https://github.com/practable/bookjs) templates the config url in to a url parameter like this:
```
https://<some-ui>?config=https%3A%2F%2Fassets.practable.io%2Fconfig%2Fexperiments%2Fpvna%2Fpvna00-0.0.json&streams=<snip>
```
