# Machine Examples

Higher level examples using the machine library.

- [Machine Examples](#machine-examples)
  * [gRPC Bidirectional Chat Server](#grpc-bidirectional-chat-server)
    + [Design](#design)
    + [Start Server](#start-server)
    + [Start Client](#start-client)

## gRPC Bidirectional Chat Server

### Design

- clients join a channel with a single request that starts streaming from os.Stdin -> chat server
- messages sent to a channel are broadcasted to all client's that are listening to that channel
- [Protobuf API](chat/chat.proto)
- [Implementation](chat/chat.go)

### Start Server

    go run chat/server/main.go --port 9000


### Start Client
In one terminal:

    go run chat/client/main.go --channel accounting --target localhost:9000
    
In a second terminal:

    go run chat/client/main.go --channel accounting --target localhost:9000

now start typing in either terminal, messages should be broadcasted to both terminals since they are both 
subscribed to the accounting channel.
    