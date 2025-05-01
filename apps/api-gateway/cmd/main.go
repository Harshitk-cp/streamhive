package main

import (
	"context"
	"log"
	"net"

	pb "github.com/Harshitk-cp/streamhive/libs/proto/signaling"

	"google.golang.org/grpc"
)

type server struct {
	pb.SignalingServiceServer
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterSignalingServiceServer(s, &server{})
	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *server) SendSignal(ctx context.Context, req *pb.SignalRequest) (*pb.SignalResponse, error) {
	log.Printf("Received signal: SessionID=%s Payload=%s", req.GetSessionId(), req.GetPayload())

	return &pb.SignalResponse{
		Status: "Received",
	}, nil
}
