package transport

import (
	"context"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryInterceptorFunc is a function that intercepts unary RPC calls
type UnaryInterceptorFunc func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error)

// StreamInterceptorFunc is a function that intercepts streaming RPC calls
type StreamInterceptorFunc func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error

// ChainUnaryInterceptors chains multiple unary interceptors into one
func ChainUnaryInterceptors(interceptors ...UnaryInterceptorFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		buildChain := func(current UnaryInterceptorFunc, next grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return current(currentCtx, currentReq, info, next)
			}
		}

		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}

		return chain(ctx, req)
	}
}

// ChainStreamInterceptors chains multiple stream interceptors into one
func ChainStreamInterceptors(interceptors ...StreamInterceptorFunc) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		buildChain := func(current StreamInterceptorFunc, next grpc.StreamHandler) grpc.StreamHandler {
			return func(currentSrv interface{}, currentStream grpc.ServerStream) error {
				return current(currentSrv, currentStream, info, next)
			}
		}

		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}

		return chain(srv, ss)
	}
}

// LoggingInterceptor logs information about unary gRPC calls
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	// Extract client metadata
	md, _ := metadata.FromIncomingContext(ctx)

	// Extract client IP from metadata
	clientIP := "unknown"
	if ips, ok := md["x-forwarded-for"]; ok && len(ips) > 0 {
		clientIP = ips[0]
	} else if ips, ok := md["x-real-ip"]; ok && len(ips) > 0 {
		clientIP = ips[0]
	}

	// Log request start
	log.Printf("gRPC Request: method=%s client_ip=%s", info.FullMethod, clientIP)

	// Call the handler
	resp, err := handler(ctx, req)

	// Calculate duration
	duration := time.Since(start)

	// Determine status code
	statusCode := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			statusCode = st.Code()
		} else {
			statusCode = codes.Unknown
		}
	}

	// Log request completion
	log.Printf("gRPC Response: method=%s client_ip=%s status=%s duration=%s",
		info.FullMethod, clientIP, statusCode, duration)

	return resp, err
}

// StreamLoggingInterceptor logs information about streaming gRPC calls
func StreamLoggingInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()

	// Extract client metadata
	ctx := ss.Context()
	md, _ := metadata.FromIncomingContext(ctx)

	// Extract client IP from metadata
	clientIP := "unknown"
	if ips, ok := md["x-forwarded-for"]; ok && len(ips) > 0 {
		clientIP = ips[0]
	} else if ips, ok := md["x-real-ip"]; ok && len(ips) > 0 {
		clientIP = ips[0]
	}

	// Log stream start
	log.Printf("gRPC Stream Start: method=%s client_ip=%s is_client_stream=%t is_server_stream=%t",
		info.FullMethod, clientIP, info.IsClientStream, info.IsServerStream)

	// Call the handler
	err := handler(srv, ss)

	// Calculate duration
	duration := time.Since(start)

	// Determine status code
	statusCode := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			statusCode = st.Code()
		} else {
			statusCode = codes.Unknown
		}
	}

	// Log stream end
	log.Printf("gRPC Stream End: method=%s client_ip=%s status=%s duration=%s",
		info.FullMethod, clientIP, statusCode, duration)

	return err
}

// RecoveryInterceptor recovers from panics in unary gRPC calls
func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in %s: %v\n%s", info.FullMethod, r, debug.Stack())
			err := status.Errorf(codes.Internal, "Internal server error")
			panic(err) // Re-panic with gRPC error
		}
	}()

	return handler(ctx, req)
}

// StreamRecoveryInterceptor recovers from panics in streaming gRPC calls
func StreamRecoveryInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in %s: %v\n%s", info.FullMethod, r, debug.Stack())
			err := status.Errorf(codes.Internal, "Internal server error")
			panic(err) // Re-panic with gRPC error
		}
	}()

	return handler(srv, ss)
}

// AuthInterceptor checks authentication for unary gRPC calls
func AuthInterceptor(authService *service.AuthService) UnaryInterceptorFunc {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authentication for certain methods
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "Metadata is not provided")
		}

		// Check for authorization header
		authHeader, ok := md["authorization"]
		if !ok || len(authHeader) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "Authorization header is not provided")
		}

		// Extract token from authorization header
		token := authHeader[0]
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
		}

		// Add claims to context
		newCtx := context.WithValue(ctx, "user_id", claims.UserID)
		newCtx = context.WithValue(newCtx, "role", claims.Role)

		// Call the handler with the new context
		return handler(newCtx, req)
	}
}

// StreamAuthInterceptor checks authentication for streaming gRPC calls
func StreamAuthInterceptor(authService *service.AuthService) StreamInterceptorFunc {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication for certain methods
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		// Extract token from metadata
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Errorf(codes.Unauthenticated, "Metadata is not provided")
		}

		// Check for authorization header
		authHeader, ok := md["authorization"]
		if !ok || len(authHeader) == 0 {
			return status.Errorf(codes.Unauthenticated, "Authorization header is not provided")
		}

		// Extract token from authorization header
		token := authHeader[0]
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
		}

		// Wrap the stream with a new context containing claims
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx: context.WithValue(
				context.WithValue(ctx, "user_id", claims.UserID),
				"role", claims.Role,
			),
		}

		// Call the handler with the wrapped stream
		return handler(srv, wrappedStream)
	}
}

// RateLimitInterceptor applies rate limiting to unary gRPC calls
func RateLimitInterceptor(rateLimiter *service.RateLimiter) UnaryInterceptorFunc {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip rate limiting for certain methods
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract client ID for rate limiting
		clientID := extractClientID(ctx)

		// Check rate limit
		err := rateLimiter.Allow(clientID)
		if err != nil {
			return nil, status.Errorf(codes.ResourceExhausted, "Rate limit exceeded: %v", err)
		}

		// Call the handler
		return handler(ctx, req)
	}
}

// StreamRateLimitInterceptor applies rate limiting to streaming gRPC calls
func StreamRateLimitInterceptor(rateLimiter *service.RateLimiter) StreamInterceptorFunc {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip rate limiting for certain methods
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		// Extract client ID for rate limiting
		clientID := extractClientID(ss.Context())

		// Check rate limit
		err := rateLimiter.Allow(clientID)
		if err != nil {
			return status.Errorf(codes.ResourceExhausted, "Rate limit exceeded: %v", err)
		}

		// Call the handler
		return handler(srv, ss)
	}
}

// wrappedServerStream wraps a grpc.ServerStream with a new context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// extractClientID extracts a client identifier for rate limiting
func extractClientID(ctx context.Context) string {
	// Try to get user_id from context
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		return userID
	}

	// If no user_id, try to get from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "unknown"
	}

	// Try to get from x-forwarded-for or x-real-ip
	if ips, ok := md["x-forwarded-for"]; ok && len(ips) > 0 {
		return ips[0]
	}
	if ips, ok := md["x-real-ip"]; ok && len(ips) > 0 {
		return ips[0]
	}

	// Fallback to unknown
	return "unknown"
}

// isPublicMethod checks if a method is public (doesn't require authentication)
func isPublicMethod(method string) bool {
	// List of public methods that don't require authentication
	publicMethods := []string{
		"/signaling.SignalingService/WebRTCConnect",  // Allow initial connections
		"/grpc.health.v1.Health/Check",               // Health checks
		"/grpc.reflection.v1alpha.ServerReflection/", // Server reflection
	}

	// Check if method is in the list
	for _, publicMethod := range publicMethods {
		if strings.HasPrefix(method, publicMethod) {
			return true
		}
	}

	return false
}
