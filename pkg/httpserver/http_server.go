package httpserver

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/voi-oss/svc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultHTTPPort        = 8080
	DefaultShutdownTimeout = time.Second * 30
)

var _ svc.Worker = (*HTTPServer)(nil)

type HTTPServer struct {
	logger *zap.Logger
	router http.Handler
	server *http.Server

	listen          string
	shutDownTimeout time.Duration
}

func New(handler http.Handler, ops ...Option) *HTTPServer {
	logger := zap.L()
	defaultErrLogger, err := zap.NewStdLogAt(logger, zapcore.ErrorLevel)
	if err != nil {
		defaultErrLogger = nil
	}

	httpSRV := &HTTPServer{
		logger: logger,
		router: handler,
		server: &http.Server{
			Addr:     net.JoinHostPort("", strconv.Itoa(DefaultHTTPPort)),
			Handler:  handler,
			ErrorLog: defaultErrLogger,
		},
		shutDownTimeout: DefaultShutdownTimeout,
	}
	for i, o := range ops {
		httpSRV.logger.Info("Running opt ", zap.Int("index", i))
		o(httpSRV)
	}
	return httpSRV
}

func (httpSRV *HTTPServer) Init(logger *zap.Logger) error {
	if logger == nil {
		return nil
	}

	httpSRV.logger = logger
	defaultErrLogger, err := zap.NewStdLogAt(logger, zapcore.ErrorLevel)
	if err != nil {
		return err
	}

	httpSRV.server.ErrorLog = defaultErrLogger
	return nil
}

func (httpSRV *HTTPServer) Run() error {
	httpSRV.logger.Info("listening to " + httpSRV.listen)
	return httpSRV.server.ListenAndServe()
}

func (httpSRV *HTTPServer) Terminate() error {
	ctx, cancel := context.WithTimeout(context.Background(), httpSRV.shutDownTimeout)
	defer cancel()
	return httpSRV.server.Shutdown(ctx)
}

func (httpSRV *HTTPServer) Healthy() error {
	return nil
}
