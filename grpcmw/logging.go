package grpcmw

import (
	"context"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryClientLogging returns a gRPC unary client interceptor that logs
// outgoing requests and their results using zap.
//
// Fields:
//   - grpc_type: "unary"
//   - grpc_method
//   - grpc_code
//   - duration
//   - error (when non-nil)
func UnaryClientLogging(logger *zap.Logger) grpc.UnaryClientInterceptor {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		dur := time.Since(start)

		st, _ := status.FromError(err)
		code := codes.OK
		if st != nil {
			code = st.Code()
		}

		fields := []zap.Field{
			zap.String("grpc_type", "unary"),
			zap.String("grpc_method", method),
			zap.String("grpc_code", code.String()),
			zap.Duration("duration", dur),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("grpc client call failed", fields...)
			return err
		}

		logger.Info("grpc client call", fields...)
		return nil
	}
}

// StreamClientLogging returns a gRPC streaming client interceptor that logs
// outgoing streaming RPCs and their results using zap.
func StreamClientLogging(logger *zap.Logger) grpc.StreamClientInterceptor {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		start := time.Now()
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		dur := time.Since(start)

		st, _ := status.FromError(err)
		code := codes.OK
		if st != nil {
			code = st.Code()
		}

		fields := []zap.Field{
			zap.String("grpc_type", "stream"),
			zap.String("grpc_method", method),
			zap.String("grpc_code", code.String()),
			zap.Duration("duration", dur),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("grpc client stream failed", fields...)
			return nil, err
		}

		logger.Info("grpc client stream", fields...)
		return clientStream, nil
	}
}

// UnaryClientRecovery returns a gRPC unary client interceptor that converts
// panics into gRPC errors and logs them with stack traces.
func UnaryClientRecovery(logger *zap.Logger) grpc.UnaryClientInterceptor {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.Error("panic recovered in grpc unary client",
					zap.String("grpc_method", method),
					zap.Any("panic", r),
					zap.ByteString("stack", stack),
				)
				err = status.Errorf(codes.Internal, "internal client panic")
			}
		}()

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientRecovery returns a gRPC streaming client interceptor that converts
// panics into gRPC errors and logs them with stack traces.
func StreamClientRecovery(logger *zap.Logger) grpc.StreamClientInterceptor {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (clientStream grpc.ClientStream, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.Error("panic recovered in grpc stream client",
					zap.String("grpc_method", method),
					zap.Any("panic", r),
					zap.ByteString("stack", stack),
				)
				err = status.Errorf(codes.Internal, "internal client panic")
				clientStream = nil
			}
		}()

		return streamer(ctx, desc, cc, method, opts...)
	}
}
