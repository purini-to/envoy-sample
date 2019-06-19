package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/purini-to/zapmw"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var TraceHeadersToPropagate = []string{
	"X-Ot-Span-Context",
	"X-Request-Id",

	// Zipkin headers
	"X-B3-TraceId",
	"X-B3-SpanId",
	"X-B3-ParentSpanId",
	"X-B3-Sampled",
	"X-B3-Flags",

	// Jaeger header (for native client)
	"uber-trace-id",
}

var (
	port            int
	debug           bool
	gstimeout       time.Duration
	name            = os.Getenv("SERVICE_NAME")
	privateEndpoint = os.Getenv("PRIVATE_ENDPOINT")

	client = &http.Client{}
)

func main() {
	flag.IntVar(&port, "port", 8080,
		`It is a port to listen for HTTP`,
	)
	flag.BoolVar(&debug, "debug", false,
		`Flag to run in the debug environment`,
	)
	flag.DurationVar(&gstimeout, "gstimeout", time.Second*30,
		`It is wait time for graceful shutdown`,
	)
	flag.Parse()

	run()
}

func run() {
	var (
		logger *zap.Logger
		err    error
	)

	if debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		panic(fmt.Sprintf("Logger initialize failed. err: %s", err.Error()))
	}

	r := chi.NewRouter()

	r.Use(Middleware(logger)...)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		host, _ := os.Hostname()
		ifaces, _ := net.Interfaces()
		addr, _ := ifaces[0].Addrs()
		ip := addr[0].String()
		w.Write([]byte(fmt.Sprintf("name: %s, host: %s, ip: %v", name, host, ip)))
	})

	r.Get("/private", func(w http.ResponseWriter, r *http.Request) {
		req, _:= http.NewRequest("GET", privateEndpoint, nil)

		for _, h := range TraceHeadersToPropagate {
			val := r.Header.Get(h)
			if len(val) > 0 {
				req.Header.Set(h, val)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		w.Write([]byte(fmt.Sprintf(`private api call. response: [%s]`, body)))
	})

	p := fmt.Sprintf(":%d", port)
	s := http.Server{Addr: p, Handler: r}

	go func() {
		logger.Info("Listen and serve", zap.String("transport", "HTTP"), zap.String("port", p))
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatal("Listen failed", zap.Error(err))
		}
	}()

	sig := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		sig <- fmt.Errorf("%s", <-c)
	}()

	logger.Info(fmt.Sprintf("SIGNAL %v received, then shutting down...", <-sig), zap.Duration("timeout", gstimeout))
	ctx, cancel := context.WithTimeout(context.Background(), gstimeout)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		// Error from closing listeners, or context timeout:
		logger.Error("Failed to gracefully shutdown", zap.Error(err))
	}

	logger.Info("Exit")
}

func Middleware(logger *zap.Logger) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.RealIP,
		zapmw.WithZap(logger, withReqId),
		zapmw.Request(zapcore.InfoLevel, "request"),
		zapmw.Recoverer(zapcore.ErrorLevel, "recover", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		}),
	}
}


func withReqId(logger *zap.Logger, r *http.Request) *zap.Logger {
	reqId := r.Header.Get("X-Request-Id")
	if len(reqId) > 0 {
		return logger.With(zap.String("reqId", reqId))
	}
	return logger
}
