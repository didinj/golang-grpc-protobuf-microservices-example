package main

import (
	"context"
	"log"
	"net"
	"sync"

	pb "github.com/didinj/grpc-go-microservices/gen/proto/user" // Import the generated User proto package
	"github.com/google/uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
	mu    sync.RWMutex
	users map[string]*pb.User
}

func newUserServer() *userServer {
	return &userServer{
		users: make(map[string]*pb.User),
	}
}

func (s *userServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if req.GetName() == "" || req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "name and email are required")
	}

	id := uuid.NewString()
	u := &pb.User{
		Id:    id,
		Name:  req.GetName(),
		Email: req.GetEmail(),
	}

	s.mu.Lock()
	s.users[id] = u
	s.mu.Unlock()

	return &pb.CreateUserResponse{User: u}, nil
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	s.mu.RLock()
	u, ok := s.users[req.GetId()]
	s.mu.RUnlock()

	if !ok {
		return nil, status.Errorf(codes.NotFound, "user %s not found", req.GetId())
	}

	return &pb.GetUserResponse{User: u}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, newUserServer())

	// Enable reflection
	reflection.Register(s)

	log.Println("✅ User service listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
