package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/autom8ter/machine"
	"github.com/autom8ter/machine/examples/helpers"
	"go.uber.org/zap"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	flag.IntVar(&port, "port", 5000, "local port to listen on")
	flag.StringVar(&target, "target", "", "target to proxy to ex: localhost:3000")
	flag.Parse()
}

var (
	logger = helpers.Logger(zap.String("example", "reverse-proxy"))
	port   int
	target string
)

func main() {
	if target == "" {
		logger.Warn("empty target, please specify with --target")
		return
	}
	targetAddr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		logger.Error("failed to resolve target", zap.Error(err))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mach := machine.New(ctx,
		machine.WithTags([]string{"reverse-proxy"}),
	)

	pending := make(chan net.Conn, 100)
	mach.Go(func(routine machine.Routine) {
		for {
			select {
			case <-routine.Context().Done():
				return
			case c := <-pending:
				if routine.Context().Err() != nil {
					return
				}
				mach.Go(func(routine machine.Routine) {
					targetConn, err := net.DialTCP("tcp", nil, targetAddr)
					if err != nil {
						logger.Error("failed to dial target", zap.Error(err))
						return
					}
					targetConn.SetWriteBuffer(1024)
					targetConn.SetReadBuffer(1024)
					routine.Machine().Go(func(routine machine.Routine) {
						for {
							select {
							case <-routine.Context().Done():
								targetConn.Close()
								c.Close()
								return
							default:
								_, err := io.CopyN(c, targetConn, 1024)
								if err != nil {
									if err == io.EOF {
										return
									} else {
										logger.Error("streaming to target error", zap.Error(err))
										continue
									}
								}
							}
						}
					}, machine.GoWithTags("stream-from-client-to-target"))
					routine.Machine().Go(func(routine machine.Routine) {
						for {
							select {
							case <-routine.Context().Done():
								targetConn.Close()
								c.Close()
								return
							default:
								_, err := io.CopyN(targetConn, c, 1024)
								if err == io.EOF {
									return
								} else {
									logger.Error("streaming from target error", zap.Error(err))
									continue
								}
							}
						}
					}, machine.GoWithTags("stream-from-target-to-client"))
				}, machine.GoWithTags("stream"))
			}
		}
	})
	httpLis, err := net.Listen("tcp", fmt.Sprintf(":%v", port+1))
	if err != nil {
		logger.Error("failed to listen http", zap.Error(err))
		return
	}
	tcpLis, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		logger.Error("failed to listen tcp", zap.Error(err))
		return
	}
	httpServer := &http.Server{Handler: http.DefaultServeMux}
	mach.Go(func(routine machine.Routine) {
		logger.Info("starting http server",
			zap.String("addr", fmt.Sprintf(":%v", port+1)),
		)
		if err := httpServer.Serve(httpLis); err != nil && err != http.ErrServerClosed {
			logger.Warn("http server failure", zap.Error(err))
		}
	}, machine.GoWithTags("http-server"))

	mach.Go(func(routine machine.Routine) {
		logger.Info("starting tcp server",
			zap.String("addr", tcpLis.Addr().String()),
			zap.String("target", target),
		)
		for {
			select {
			case <-routine.Context().Done():
				return
			default:
				conn, err := tcpLis.Accept()
				if err != nil {
					if opErr, ok := err.(*net.OpError); ok {
						if opErr.Timeout() {
							continue
						}
					}
					if routine.Context().Err() == nil {
						logger.Error("tcp listener error", zap.Error(err))
					}
					return
				}
				if routine.Context().Err() == nil {
					pending <- conn
				}
			}
		}
	}, machine.GoWithTags("tcp-server"))
	interrupt := make(chan os.Signal, 1)

	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	select {
	case <-interrupt:
		mach.Cancel()
		break
	case <-ctx.Done():
		break
	}
	logger.Error("shutdown signal received")
	cancel()
	tcpLis.Close()
	httpLis.Close()
	mach.Wait()
}
