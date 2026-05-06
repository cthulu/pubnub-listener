```
docker build -t nl.freshlytyped.nl/pubnub .
docker run -e PN_PUBLISH_KEY= \
    -e PN_SUBSCRIBE_KEY= \
    -e BEARER_TOKEN= \
    nl.freshlytyped.nl/pubnub 
```
