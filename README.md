# crypto_analytics
Golang + kafka based analytics pipeline for crypto analytics






# debug kafka broker
Displays live stream of events from producer
```
docker exec -it $(docker ps -qf "ancestor=apache/kafka:3.7.0") /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic trades --from-beginning
```


simple demo with web ui [https://crypto.r-tan.dev/](https://crypto.r-tan.dev/)

