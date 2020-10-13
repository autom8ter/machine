# Machine Examples

Higher level examples using the machine library.

- [Machine Examples](#machine-examples)
  * [gRPC Bidirectional Chat Server](#grpc-bidirectional-chat-server)
    + [Design](#design)
    + [Start Server](#start-server)
    + [Start Client](#start-client)

## gRPC Bidirectional Chat Server
- (~ 300 lines)
### Design

- clients join a channel with a single request that starts streaming from os.Stdin -> chat server
- messages sent to a channel are broadcasted to all client's that are listening to that channel
- [Protobuf API](chat/chat.proto)
- [Implementation](chat/chat.go)

#### Start Server

    go run chat/server/main.go --port 9000


#### Start Client
In one terminal:

    go run chat/client/main.go --channel accounting --target localhost:9000
    
In a second terminal:

    go run chat/client/main.go --channel accounting --target localhost:9000

now start typing in either terminal, messages should be broadcasted to both terminals since they are both 
subscribed to the accounting channel.

## TCP Reverse Proxy

- [Implementation](reverse-proxy/main.go)

- (~ 150 lines)
- straight up transparent reverse proxy
- zero dependencies besides logger


    go run reverse-proxy/main.go --port 5000 --target $(upstream ip address/port)

Test proxy:

    curl $(local proxy)

## Concurrent Cron Server

- [Implementation](cron/main.go)

- (~ 150 lines)
- zero dependencies besides logger
- execute any number of shell scripts in separate threads, each with their own timer
- config file to hold cron job configuration
- every cron is cleaned up when the parent goroutine is cancelled/times out
- get crons via http GET request to /cron
- add cron at runtime via http POST request to /cron/job


example config: 
```text
name: "example"
jobs:
  - name: "hello world"
    sleep: "1s"
    script: |
      echo "hello world"

  - name: "hello world 2"
    sleep: "1s"
    script: |
      echo "hello world 2"
```

#### Start Cron Server
    
    go run cron/main.go --config cron/cron.yaml --port 8000

#### Get Cron Jobs

    curl -X GET http://localhost:8000/cron
    
#### Add Cron Job

    curl -L -X POST 'http://localhost:8000/cron/job' \
    -H 'Content-Type: application/json' \
    --data-raw '{
        "name": "whoami",
        "script": "whoami",
        "sleep": "5s"
    }'