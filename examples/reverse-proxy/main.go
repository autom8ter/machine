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
		logger.Warn("failed to resolve target", zap.Error(err))
		return
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		logger.Warn("failed to listen", zap.Error(err))
		return
	}
	defer lis.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mach := machine.New(ctx)

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
						logger.Warn("failed to dial target", zap.Error(err))
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
										logger.Warn("streaming to target error", zap.Error(err))
										continue
									}
								}
							}
						}
					})
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
									logger.Warn("streaming from target error", zap.Error(err))
									continue
								}
							}
						}
					})
				})
			}
		}
	})
	logger.Info("starting tcp listener",
		zap.String("addr", lis.Addr().String()),
		zap.String("target", target),
	)
	mach.Go(func(routine machine.Routine) {
		for {
			select {
			case <-routine.Context().Done():
				return
			default:
				conn, err := lis.Accept()
				if err != nil {
					if opErr, ok := err.(*net.OpError); ok {
						if opErr.Timeout() {
							continue
						}
					}
					if routine.Context().Err() == nil {
						logger.Warn("tcp listener error", zap.Error(err))
					}
					return
				}
				if routine.Context().Err() == nil {
					pending <- conn
				}
			}
		}
	})
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
	logger.Warn("shutdown signal received")
	cancel()
	lis.Close()
	mach.Wait()
}
