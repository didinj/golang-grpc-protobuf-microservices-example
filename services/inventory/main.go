package main

import (
	"context"
	"log"
	"net"
	"sync"

	pb "github.com/didinj/grpc-go-microservices/gen/proto/inventory" // 👈 matches option go_package = "gen/inventory;inventory"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type inventoryServer struct {
	pb.UnimplementedInventoryServiceServer
	mu    sync.RWMutex
	items map[string]*pb.Item
}

func newInventoryServer() *inventoryServer {
	return &inventoryServer{
		items: make(map[string]*pb.Item),
	}
}

func (s *inventoryServer) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.GetQuantity() < 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be >= 0")
	}

	id := uuid.NewString()
	item := &pb.Item{
		Id:       id,
		Name:     req.GetName(),
		Quantity: req.GetQuantity(),
	}

	s.mu.Lock()
	s.items[id] = item
	s.mu.Unlock()

	return &pb.CreateItemResponse{Item: item}, nil
}

func (s *inventoryServer) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.GetItemResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	s.mu.RLock()
	item, ok := s.items[req.GetId()]
	s.mu.RUnlock()

	if !ok {
		return nil, status.Errorf(codes.NotFound, "item %s not found", req.GetId())
	}

	return &pb.GetItemResponse{Item: item}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50052") // 👈 runs on a different port than UserService
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterInventoryServiceServer(s, newInventoryServer())

	// Enable reflection
	reflection.Register(s)

	log.Println("✅ Inventory service listening on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
