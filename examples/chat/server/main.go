package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/autom8ter/machine"
	"github.com/autom8ter/machine/examples/chat"
	chatpb "github.com/autom8ter/machine/examples/gen/go/example/chat"
	"github.com/autom8ter/machine/examples/helpers"
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
	m := machine.New(ctx)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		panic(err)
	}
	defer lis.Close()
	srv := grpc.NewServer()
	logger := helpers.Logger(zap.String("type", "server"))
	chatpb.RegisterChatServiceServer(srv, chat.NewChatServer(logger, m))
	m.Go(func(routine machine.Routine) {
		logger.Info("starting server", zap.String("addr", lis.Addr().String()))
		if err := srv.Serve(lis); err != nil {
			logger.Warn("server failure", zap.Error(err))
			return
		}
	})
	m.Go(func(routine machine.Routine) {
		for {
			select {
			case <-routine.Context().Done():
				logger.Info("shutting down server")
				srv.Stop()
			}
		}
	})
	m.Wait()
}
