package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var requestSequence uint64

func serveAddress(address string, handler http.Handler, shutdown func()) error {
	return serveHTTP(newEnterpriseServer(address, handler), shutdown)
}

func serveHTTP(server *http.Server, shutdown func()) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-errCh:
		// Server failed on its own (e.g. bad bind). Stop background tasks so the
		// process does not leak goroutines before exiting.
		if shutdown != nil {
			shutdown()
		}
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals:
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := server.Shutdown(shutdownContext)
		// Drain background tasks after the HTTP server stops, so the process
		// exits cleanly instead of hanging on a leaked ticker goroutine.
		if shutdown != nil {
			shutdown()
		}
		return err
	}
}

func newEnterpriseServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           opsEnterpriseMiddleware(requestIDMiddleware(recoveryMiddleware(handler))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", atomic.AddUint64(&requestSequence, 1))
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered request_id=%s method=%s path=%s panic=%v", w.Header().Get("X-Request-ID"), r.Method, r.URL.Path, recovered)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
