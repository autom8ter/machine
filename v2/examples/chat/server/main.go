package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/autom8ter/machine/v2"
	"github.com/autom8ter/machine/v2/examples/chat"
	chatpb "github.com/autom8ter/machine/v2/examples/gen/go/example/chat"
	"github.com/autom8ter/machine/v2/examples/helpers"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
)

func init() {
	flag.IntVar(&port, "port", 9000, "port to serve on")
	flag.Parse()
}

var (
	port int
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := machine.New()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		panic(err)
	}
	defer lis.Close()
	srv := grpc.NewServer()
	logger := helpers.Logger(zap.String("type", "server"))
	chatpb.RegisterChatServiceServer(srv, chat.NewChatServer(logger, m))
	logger.Info("starting server", zap.String("addr", lis.Addr().String()))
	m.Go(ctx, func(ctx context.Context) error {
		if err := srv.Serve(lis); err != nil {
			return err
		}
		return nil
	})
	m.Go(ctx, func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Info("shutting down server")
				srv.Stop()
			}
		}
	})
	m.Wait()
}
