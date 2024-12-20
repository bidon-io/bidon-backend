package config

import (
	"context"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewGRPCServer creates a new gRPC server with all middleware configured
func NewGRPCServer(logger *zap.Logger) *grpc.Server {
	metrics := NewMetrics()
	rpcLogger := logger.Named("grpc_server").Sugar()

	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			metrics.UnaryServerInterceptor(),
			logging.UnaryServerInterceptor(
				zapToGRPCLoggerAdapter(rpcLogger),
				logging.WithFieldsFromContext(logTraceID),
			),
			unaryErrorInterceptor(rpcLogger),
			recovery.UnaryServerInterceptor(
				recovery.WithRecoveryHandler(
					newPanicRecoveryHandler(metrics.Registry, rpcLogger),
				),
			),
		),
		grpc.ChainStreamInterceptor(
			metrics.StreamServerInterceptor(),
			logging.StreamServerInterceptor(
				zapToGRPCLoggerAdapter(rpcLogger),
				logging.WithFieldsFromContext(logTraceID),
			),
			streamErrorInterceptor(rpcLogger),
			recovery.StreamServerInterceptor(
				recovery.WithRecoveryHandler(
					newPanicRecoveryHandler(metrics.Registry, rpcLogger),
				),
			),
		),
	)

	metrics.ServerMetrics.InitializeMetrics(srv)

	return srv
}

type Metrics struct {
	Registry      *prometheus.Registry
	ServerMetrics *grpcprom.ServerMetrics
}

// NewMetrics creates a new metrics instance with configured collectors
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	reg.MustRegister(srvMetrics)

	return &Metrics{
		Registry:      reg,
		ServerMetrics: srvMetrics,
	}
}

// UnaryServerInterceptor returns a configured unary interceptor for metrics
func (m *Metrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return m.ServerMetrics.UnaryServerInterceptor(
		grpcprom.WithExemplarFromContext(exemplarFromContext),
	)
}

// StreamServerInterceptor returns a configured stream interceptor for metrics
func (m *Metrics) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return m.ServerMetrics.StreamServerInterceptor(
		grpcprom.WithExemplarFromContext(exemplarFromContext),
	)
}

func unaryErrorInterceptor(logger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		resp, err = handler(ctx, req)

		if err != nil {
			logger.Errorw("gRPC error",
				"method", info.FullMethod,
				"error", err,
			)

			// Use the current hub to capture the error in Sentry
			hub := sentry.CurrentHub()
			if hub != nil {
				hub.Scope().SetContext("gRPC", map[string]interface{}{
					"method": info.FullMethod,
					"error":  err.Error(),
				})
				hub.CaptureException(err)
			}
		}

		return resp, err
	}
}

func streamErrorInterceptor(logger *zap.SugaredLogger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, stream)

		if err != nil {
			logger.Errorw("gRPC stream error",
				"method", info.FullMethod,
				"error", err,
				"isClientStream", info.IsClientStream,
				"isServerStream", info.IsServerStream,
			)

			hub := sentry.CurrentHub()
			if hub != nil {
				hub.Scope().SetContext("gRPC", map[string]interface{}{
					"method":         info.FullMethod,
					"error":          err.Error(),
					"isClientStream": info.IsClientStream,
					"isServerStream": info.IsServerStream,
				})
				hub.CaptureException(err)
			}
		}

		return err
	}
}

func newPanicRecoveryHandler(reg prometheus.Registerer, logger *zap.SugaredLogger) func(p any) error {
	panicsTotal := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "grpc_req_panics_recovered_total",
		Help: "Total number of gRPC requests recovered from internal panic.",
	})

	return func(p any) error {
		panicsTotal.Inc()

		logger.Errorw("recovered from panic",
			"panic", p,
			"stack", string(debug.Stack()),
		)

		sentry.CurrentHub().Recover(p)

		return status.Errorf(codes.Internal, "%s", p)
	}
}

// exemplarFromContext extracts trace ID from context for Prometheus exemplars
func exemplarFromContext(ctx context.Context) prometheus.Labels {
	if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
		return prometheus.Labels{"traceID": span.TraceID().String()}
	}
	return nil
}

// zapToGRPCLoggerAdapter converts zap logger to grpc logger
func zapToGRPCLoggerAdapter(l *zap.SugaredLogger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		switch lvl {
		case logging.LevelDebug:
			l.Debugw(msg, fields...)
		case logging.LevelInfo:
			l.Infow(msg, fields...)
		case logging.LevelWarn:
			l.Warnw(msg, fields...)
		case logging.LevelError:
			l.Errorw(msg, fields...)
		default:
			l.Infow(msg, fields...)
		}
	})
}

// LogTraceID extracts trace ID from context for logging
func logTraceID(ctx context.Context) logging.Fields {
	if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
		return logging.Fields{"traceID", span.TraceID().String()}
	}
	return nil
}
