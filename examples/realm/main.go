package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// main parses the flags, builds the server and runs it until interrupted. It holds
// no logic of the realm: that is server.go and game.go.
func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address to listen on")
	seed := flag.Uint64("seed", 1, "seed of the world")
	pace := flag.Duration("pace", 0, "step the world once every pace (0: only POST /v1/step steps it)")
	state := flag.String("state", "", "checkpoint file: restored at start if it exists, written by POST /v1/checkpoint?save=true")
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(2)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Listening first fails a bad address before the start-up solve; connections made
	// meanwhile wait in the backlog until serve accepts them.
	l, err := net.Listen("tcp", *addr)
	if err == nil {
		err = serve(ctx, logger, l, *seed, *pace, *state)
	}
	if err != nil {
		logger.Error("realm stopped", slog.Any("error", err))
		stop()
		os.Exit(1)
	}
}

// serve runs the server on l until ctx is done, and closes l. There is no write
// timeout: a POST /v1/step of 1 000 ticks answers after about a minute.
func serve(ctx context.Context, logger *slog.Logger, l net.Listener, seed uint64, pace time.Duration, state string) error {
	if pace < 0 { // 0 is "off"; a negative value is a typo, not a request
		return errors.Join(fmt.Errorf("pace %v: %w", pace, ErrBadInput), l.Close())
	}
	s, err := NewServer(seed, state, logger)
	if err != nil {
		return errors.Join(err, l.Close())
	}
	addr := l.Addr().String()
	// Bound to one host, only the names a local client uses and that host may reach the server.
	hosts := []string{"localhost", "127.0.0.1", "::1"}
	if h, _, err := net.SplitHostPort(addr); err == nil && h != "" {
		hosts = append(hosts, h)
	}
	if h, _, err := net.SplitHostPort(addr); err != nil || (h != "" && h != "0.0.0.0" && h != "::") {
		s.AllowHosts(hosts...)
	} // -addr :8080 listens on every interface: any Host may be a legitimate name of this machine
	if pace > 0 {
		clock := time.NewTicker(pace)
		defer clock.Stop()
		go s.Pace(ctx, clock.C)
	}
	srv := &http.Server{
		Handler: s,
		// In-flight steps stop at a tick boundary when the process is told to stop.
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: time.Minute, IdleTimeout: time.Minute,
	}
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(sctx); err != nil {
			logger.Error("shut the server down", slog.Any("error", err))
		}
	}()
	logger.Info("realm listening", slog.String("addr", addr), slog.Uint64("seed", seed))
	if err := srv.Serve(l); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-drained // ListenAndServe returns as soon as Shutdown starts; wait for it to finish
	return nil
}
