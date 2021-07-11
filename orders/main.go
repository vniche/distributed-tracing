package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/vniche/distributed-tracing/orders/protocol"
	"github.com/vniche/distributed-tracing/tracing"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// ordersServer is used to implement me.vniche.store.Orders
type orderServer struct {
	protocol.UnimplementedOrdersServer
}

var fakeOrders = []*protocol.Order{
	{
		Id:       uuid.New().String(),
		Product:  "b380894d-faa5-4a0b-8d34-3d90f301ba52",
		Quantity: 12,
	},
	{
		Id:       uuid.New().String(),
		Product:  "b380894d-faa5-4a0b-8d34-3d90f301ba52",
		Quantity: 26,
	},
	{
		Id:       uuid.New().String(),
		Product:  "b380894d-faa5-4a0b-8d34-3d90f301ba52",
		Quantity: 8,
	},
	{
		Id:       uuid.New().String(),
		Product:  "b380894d-faa5-4a0b-8d34-3d90f301ba52",
		Quantity: 2,
	},
	{
		Id:       uuid.New().String(),
		Product:  "b380894d-faa5-4a0b-8d34-3d90f301ba52",
		Quantity: 4,
	},
}

func (s *orderServer) GetOrders(ctx context.Context, request *protocol.GetOrdersRequest) (*protocol.GetOrdersResponse, error) {
	if request.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "product id is required")
	}

	var orders []*protocol.Order

	// run a synchronous long-running operation to emulate database operation
	func(ctx context.Context, request *protocol.GetOrdersRequest) {
		_, span := tracing.Tracer("datastore: cache").Start(ctx, "get orders")
		defer span.End()

		for _, order := range fakeOrders {
			if order.Product == request.Id {
				orders = append(orders, order)
			}
		}

		n := (rand.Intn(3-1) + 1)
		time.Sleep(time.Duration(n) * time.Second)
		span.AddEvent("fetched orders from datastore", trace.WithAttributes(
			attribute.String("product id", request.Id),
			attribute.Int("orders.length", len(orders)),
		))
	}(ctx, request)

	return &protocol.GetOrdersResponse{
		Orders: orders,
	}, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracerProvider, err := tracing.InitTracerProvider(ctx)
	if err != nil {
		fmt.Printf("unable to initialize trace provider: %+v\n", err)
	}

	ctx, span := tracing.Tracer("main").Start(ctx, "initialization")

	// Cleanly shutdown and flush telemetry when the application exits.
	defer func(ctx context.Context) {
		// Do not make the application hang when it is shutdown.
		ctx, cancel = context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			fmt.Printf("test")
			log.Fatal(err)
		}
	}(ctx)

	datastoreCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// run a synchronous long-running operation to emulate database connection
	func(ctx context.Context) {
		_, span := tracing.Tracer("datastore").Start(ctx, "connect")
		defer span.End()

		n := (rand.Intn(8-3) + 3)
		time.Sleep(time.Duration(n) * time.Second)
	}(datastoreCtx)

	span.End()

	fmt.Printf("span end\n")

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "50051"
	}

	// start gRPC server
	var lis net.Listener
	lis, err = net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}

	grcpServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			otelgrpc.UnaryServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider())),
		),
		grpc.StreamInterceptor(
			otelgrpc.StreamServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider())),
		),
	)
	reflection.Register(grcpServer)

	protocol.RegisterOrdersServer(grcpServer, &orderServer{})
	log.Printf("server listening at %v", lis.Addr())
	if err := grcpServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
