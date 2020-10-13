package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/autom8ter/machine"
	chatpb "github.com/autom8ter/machine/examples/gen/go/example/chat"
	"github.com/autom8ter/machine/examples/helpers"
	"github.com/golang/protobuf/jsonpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
	"os"
	"strings"
)

func init() {
	flag.StringVar(&channel, "channel", "default", "channel to chat in")
	flag.StringVar(&target, "target", "localhost:9000", "target to dial when running client")
	flag.StringVar(&token, "token", "", "authorization token to send")
	flag.Parse()
}

var (
	target  string
	token   string
	channel string
	encoder = &jsonpb.Marshaler{
		Indent: "    ",
	}
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := helpers.Logger(
		zap.String("type", "client"),
		zap.String("target", target),
		zap.String("channel", channel),
	)
	m := machine.New(ctx)
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		logger.Warn("failed to dial server", zap.Error(err))
		return
	}

	defer conn.Close()
	client := chatpb.NewChatServiceClient(conn)
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"X-CHANNEL":     channel,
	}))
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	stream, err := client.Chat(ctx)
	if err != nil {
		logger.Warn("failed to start chat stream", zap.Error(err))
		return
	}
	m.Go(func(routine machine.Routine) {
		reader := bufio.NewReader(os.Stdin)
		defer reader.Discard(reader.Buffered())
		for {
			select {
			case <-routine.Context().Done():
				return
			default:
				text, _ := reader.ReadString('\n')
				text = strings.TrimSpace(text)
				if len(text) > 0 {
					if err := stream.Send(&chatpb.ChatRequest{
						Text: text,
					}); err != nil {
						logger.Warn("failed to stream from os.Stdin to server", zap.Error(err))
						return
					}
				}
				reader.Reset(os.Stdin)
			}
		}
	})
	m.Go(func(routine machine.Routine) {
		for {
			select {
			case <-routine.Context().Done():
				return
			default:
				resp, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						continue
					}
					return
				}
				if resp != nil && resp.Text != "" {
					str, err := encoder.MarshalToString(resp)
					if err != nil {
						logger.Warn("failed to encode response", zap.Error(err))
						continue
					}
					fmt.Println(str)
				}
			}
		}
	})
	m.Wait()
}
