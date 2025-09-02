package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/didinj/grpc-go-microservices/gen/proto/inventory"
	invpb "github.com/didinj/grpc-go-microservices/gen/proto/inventory"
	"github.com/didinj/grpc-go-microservices/gen/proto/order"
	orderpb "github.com/didinj/grpc-go-microservices/gen/proto/order"
	"github.com/didinj/grpc-go-microservices/gen/proto/user"
	userpb "github.com/didinj/grpc-go-microservices/gen/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type orderServer struct {
	orderpb.UnimplementedOrderServiceServer
	mu     sync.RWMutex
	orders map[string]*orderpb.Order

	userClient userpb.UserServiceClient
	invClient  invpb.InventoryServiceClient
}

func newOrderServer(userClient userpb.UserServiceClient, invClient invpb.InventoryServiceClient) *orderServer {
	return &orderServer{
		orders:     make(map[string]*orderpb.Order),
		userClient: userClient,
		invClient:  invClient,
	}
}

func (s *orderServer) CreateOrder(ctx context.Context, req *order.CreateOrderRequest) (*order.CreateOrderResponse, error) {
	// 1. Check user exists
	_, err := s.userClient.GetUser(ctx, &user.GetUserRequest{Id: req.UserId})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "user not found: %v", err)
	}

	// 2. Check item exists
	itemResp, err := s.invClient.GetItem(ctx, &inventory.GetItemRequest{Id: req.ItemId})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "item not found: %v", err)
	}

	if itemResp.Item.Quantity < req.Quantity {
		return nil, status.Errorf(codes.FailedPrecondition, "not enough stock")
	}

	// 3. Reduce stock (in real app, this should be transactional)
	itemResp.Item.Quantity -= req.Quantity

	// 4. Create order
	id := fmt.Sprintf("%d", len(s.orders)+1)
	ord := &order.Order{
		Id:       id,
		UserId:   req.UserId,
		ItemId:   req.ItemId,
		Quantity: req.Quantity,
	}
	s.orders[id] = ord

	return &order.CreateOrderResponse{Order: ord}, nil
}

func (s *orderServer) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	ord, exists := s.orders[req.Id]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.Id)
	}

	return &orderpb.GetOrderResponse{Order: ord}, nil
}

func main() {
	// Dial User Service
	userConn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to user service: %v", err)
	}
	defer userConn.Close()
	userClient := user.NewUserServiceClient(userConn)

	// Dial Inventory Service
	inventoryConn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to inventory service: %v", err)
	}
	defer inventoryConn.Close()
	inventoryClient := inventory.NewInventoryServiceClient(inventoryConn)

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	order.RegisterOrderServiceServer(s, &orderServer{
		orders:     make(map[string]*order.Order),
		userClient: userClient,
		invClient:  inventoryClient,
	})

	// Enable reflection
	reflection.Register(s)

	log.Println("✅ Order service listening on :50053")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
