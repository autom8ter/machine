package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/autom8ter/machine/v2"
	"github.com/autom8ter/machine/v2/examples/helpers"
	"github.com/pkg/errors"
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

	mach := machine.New()

	pending := make(chan net.Conn, 100)
	mach.Go(ctx, func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case c := <-pending:
				if ctx.Err() != nil {
					return nil
				}
				mach.Go(ctx, func(ctx context.Context) error {
					targetConn, err := net.DialTCP("tcp", nil, targetAddr)
					if err != nil {
						return errors.Wrap(err, "failed to dial target")
					}
					targetConn.SetWriteBuffer(1024)
					targetConn.SetReadBuffer(1024)
					mach.Go(ctx, func(ctx context.Context) error {
						for {
							select {
							case <-ctx.Done():
								targetConn.Close()
								c.Close()
								return nil
							default:
								_, err := io.CopyN(c, targetConn, 1024)
								if err != nil {
									if err == io.EOF {
										return nil
									} else {
										logger.Error("streaming to target error", zap.Error(err))
										continue
									}
								}
							}
						}
					})
					mach.Go(ctx, func(ctx context.Context) error {
						for {
							select {
							case <-ctx.Done():
								targetConn.Close()
								c.Close()
								return nil
							default:
								_, err := io.CopyN(targetConn, c, 1024)
								if err == io.EOF {
									return nil
								} else {
									logger.Error("streaming from target error", zap.Error(err))
									continue
								}
							}
						}
					})
					return nil
				})
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
	logger.Info("starting http server",
		zap.String("addr", fmt.Sprintf(":%v", port+1)),
	)
	mach.Go(ctx, func(ctx context.Context) error {
		if err := httpServer.Serve(httpLis); err != nil && err != http.ErrServerClosed {
			return errors.Wrap(err, "http server failure")
		}
		return nil
	})

	mach.Go(ctx, func(ctx context.Context) error {
		logger.Info("starting tcp server",
			zap.String("addr", tcpLis.Addr().String()),
			zap.String("target", target),
		)
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				conn, err := tcpLis.Accept()
				if err != nil {
					if opErr, ok := err.(*net.OpError); ok {
						if opErr.Timeout() {
							continue
						}
					}
					return err
				}
				if ctx.Err() == nil {
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
		cancel()
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
