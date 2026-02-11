package grpc

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

// Server wraps a gRPC server with middleware support
type Server struct {
	*grpc.Server
	options       []grpc.ServerOption
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
}

// Config holds gRPC server configuration
type Config struct {
	Host string
	Port int
}

// NewServer creates a new gRPC server
func NewServer() *Server {
	return &Server{
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
	}
}

// UseUnaryInterceptor adds a unary interceptor (middleware)
func (s *Server) UseUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) {
	s.unaryInterceptors = append(s.unaryInterceptors, interceptor)
}

// UseStreamInterceptor adds a stream interceptor
func (s *Server) UseStreamInterceptor(interceptor grpc.StreamServerInterceptor) {
	s.streamInterceptors = append(s.streamInterceptors, interceptor)
}

// Build builds the gRPC server with all options and interceptors
func (s *Server) Build() {
	opts := make([]grpc.ServerOption, 0)

	// Add unary interceptors
	if len(s.unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(s.unaryInterceptors...))
	}

	// Add stream interceptors
	if len(s.streamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(s.streamInterceptors...))
	}

	// Add any additional options
	opts = append(opts, s.options...)

	s.Server = grpc.NewServer(opts...)

	// Register reflection service for grpcurl
	reflection.Register(s.Server)
}

// Start starts the gRPC server
func (s *Server) Start(config *Config) error {
	if s.Server == nil {
		s.Build()
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Printf("[gRPC] Server starting on %s", addr)
	return s.Server.Serve(listener)
}

// GracefulStop gracefully stops the gRPC server
func (s *Server) GracefulStop() {
	if s.Server != nil {
		log.Println("[gRPC] Gracefully stopping server...")
		s.Server.GracefulStop()
	}
}

// Common Interceptors

// LoggingInterceptor logs all gRPC requests
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Printf("[gRPC] Method: %s", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			log.Printf("[gRPC] Error: %v", err)
		}
		return resp, err
	}
}

// RecoveryInterceptor recovers from panics
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[gRPC] Panic recovered: %v", r)
				err = fmt.Errorf("internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// AuthInterceptor validates authentication tokens
func AuthInterceptor(validateToken func(string) (context.Context, error)) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, fmt.Errorf("missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return nil, fmt.Errorf("missing authorization token")
		}

		token := tokens[0]
		newCtx, err := validateToken(token)
		if err != nil {
			return nil, fmt.Errorf("invalid token: %w", err)
		}

		return handler(newCtx, req)
	}
}

// MetricsInterceptor tracks request metrics
func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// TODO: Integrate with Prometheus metrics
		resp, err := handler(ctx, req)
		return resp, err
	}
}

