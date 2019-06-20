package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	escontext "github.com/purini-to/envoy-sample/context"
	"github.com/purini-to/envoy-sample/services/task"
	"github.com/purini-to/zapmw"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	port      int
	debug     bool
	gstimeout time.Duration

	dataSourceName = os.Getenv("DATA_SOURCE_NAME")
)

var (
	serviceName        = "MyService"
	serviceHostPort    = "service3-sidecar:3306"
	zipkinHTTPEndpoint = os.Getenv("ZIPKIN_HTTP_ENDPOINT")
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
		panic(fmt.Sprintf("Unable to create logger. err: %s", err.Error()))
	}

	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		logger.Panic("Unable to database connect.", zap.Error(err))
	}
	defer db.Close()

	reporter := zipkinhttp.NewReporter(zipkinHTTPEndpoint)
	defer reporter.Close()

	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("task", "localhost:0")
	if err != nil {
		logger.Panic("Unable to create local endpoint.", zap.Error(err))
	}

	// initialize our tracer
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		logger.Panic("Unable to create tracer.", zap.Error(err))
	}

	taskRepo, err := task.NewRepository(db, tracer)
	if err != nil {
		logger.Panic("Unable to task.Repository.", zap.Error(err))
	}

	r := chi.NewRouter()

	r.Use(Middleware(logger, tracer)...)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		res, err := taskRepo.FindAll(r.Context())
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	})

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		res, err := taskRepo.FindByID(r.Context(), id)
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		if res == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": http.StatusText(http.StatusNotFound)})
			return
		}

		json.NewEncoder(w).Encode(res)
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

func Middleware(logger *zap.Logger, tracer *zipkin.Tracer) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.RealIP,
		zapmw.WithZap(logger, withReqId),
		zapmw.Request(zapcore.InfoLevel, "request"),
		zapmw.Recoverer(zapcore.ErrorLevel, "recover"),
		func(next http.Handler) http.Handler {
			fn := func(w http.ResponseWriter, r *http.Request) {
				// try to extract B3 Headers from upstream
				sc := tracer.Extract(b3.ExtractHTTP(r))
				ctx := escontext.WithSpanContext(r.Context(), sc)
				next.ServeHTTP(w, r.WithContext(ctx))
			}
			return http.HandlerFunc(fn)
		},
	}
}

func withReqId(logger *zap.Logger, r *http.Request) *zap.Logger {
	reqId := r.Header.Get("X-Request-Id")
	if len(reqId) > 0 {
		return logger.With(zap.String("reqId", reqId))
	}
	return logger
}
