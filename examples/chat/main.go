package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/autom8ter/machine"
	chatpb "github.com/autom8ter/machine/examples/gen/go/example/chat"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
	"net"
	"os"
	"strings"
)

func init() {
	flag.BoolVar(&serve, "serve", false, "start server")
	flag.StringVar(&target, "target", "localhost:9000", "target to dial when running client")
	flag.IntVar(&port, "port", 9000, "port to serve on")
	flag.Parse()
}

var (
	serve   bool
	target  string
	port    int
	encoder = &jsonpb.Marshaler{
		Indent: "    ",
	}
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if serve {
		m := machine.New(ctx)
		lis, err := net.Listen("tcp", fmt.Sprintf(":%v", 9000))
		defer lis.Close()
		errExit(err)
		srv := grpc.NewServer()
		m.Go(func(routine machine.Routine) {
			if err := srv.Serve(lis); err != nil {
				errPrint(err)
			}
		})
		m.Go(func(routine machine.Routine) {
			for {
				select {
				case <-routine.Context().Done():
					srv.Stop()
				}
			}
		})
		m.Wait()
	} else {
		m := machine.New(ctx)
		conn, err := grpc.DialContext(ctx, target, grpc.WithInsecure())
		errExit(err)
		defer conn.Close()
		client := chatpb.NewChatServiceClient(conn)
		stream, err := client.Chat(ctx)
		errExit(err)

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
					reader.Reset(os.Stdin)
					if len(text) > 0 {
						if err := stream.Send(&chatpb.ChatRequest{
							Channel: "default",
							Text:    text,
						}); err != nil {
							errPrint(err)
						}
					}
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
						errPrint(err)
						continue
					}
					str, err := encoder.MarshalToString(resp)
					if err != nil {
						errPrint(err)
						continue
					}
					fmt.Println(str)
				}
			}
		})
		m.Wait()
	}
}

func errExit(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func errPrint(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
