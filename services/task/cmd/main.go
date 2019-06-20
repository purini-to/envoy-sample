package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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

	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		logger.Panic("Database connection failed.", zap.Error(err))
	}
	defer db.Close()

	taskRepo := task.NewRepository(db)

	r := chi.NewRouter()

	r.Use(Middleware(logger)...)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		res, err := taskRepo.FindAll()
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	})

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		res, err := taskRepo.FindByID(id)
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

func Middleware(logger *zap.Logger) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.RealIP,
		zapmw.WithZap(logger, withReqId),
		zapmw.Request(zapcore.InfoLevel, "request"),
		zapmw.Recoverer(zapcore.ErrorLevel, "recover"),
	}
}

func withReqId(logger *zap.Logger, r *http.Request) *zap.Logger {
	reqId := r.Header.Get("X-Request-Id")
	if len(reqId) > 0 {
		return logger.With(zap.String("reqId", reqId))
	}
	return logger
}
