package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/vniche/distributed-tracing/common"
	ordersProtocol "github.com/vniche/distributed-tracing/orders/protocol"
	"github.com/vniche/distributed-tracing/products/protocol"
	"github.com/vniche/distributed-tracing/tracing"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const port = ":50051"

var ordersClient ordersProtocol.OrdersClient

// productsServer is used to implement me.vniche.store.Products
type productsServer struct {
	protocol.UnimplementedProductsServer
}

func (server *productsServer) CreateProduct(ctx context.Context, product *protocol.Product) (*common.ChangeResponse, error) {
	if product.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}

	productBytes, err := json.Marshal(product)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to parse request content to json: %s", err.Error())
	}

	// run a synchronous long-running operation to emulate database connection
	func(ctx context.Context, product *protocol.Product, productBytes []byte) {
		_, span := tracing.Tracer("datastore: cache").Start(ctx, "create product")
		defer span.End()

		n := (rand.Intn(3-1) + 1)
		time.Sleep(time.Duration(n) * time.Second)
		fmt.Printf("to cache: %+v\n", string(productBytes))
		span.AddEvent("added product to cache", trace.WithAttributes(
			attribute.String("product.Id", product.Id),
			attribute.String("product.Name", product.Name),
		))
	}(ctx, product, productBytes)

	// run a synchronous long-running operation to emulate database connection
	func(ctx context.Context, product *protocol.Product, productBytes []byte) {
		_, span := tracing.Tracer("datastore: mongo").Start(ctx, "create product")
		defer span.End()

		n := (rand.Intn(5-2) + 2)
		time.Sleep(time.Duration(n) * time.Second)
		fmt.Printf("to mongo: %+v\n", string(productBytes))
		span.AddEvent("added product to mongo", trace.WithAttributes(
			attribute.String("product.Id", product.Id),
			attribute.String("product.Name", product.Name),
		))
	}(ctx, product, productBytes)

	return &common.ChangeResponse{
		Message: "creation successfully requested",
	}, nil
}

func (server *productsServer) GetProductOrders(ctx context.Context, request *protocol.GetProductOrdersRequest) (*protocol.GetProductOrdersResponse, error) {
	resp, err := ordersClient.GetOrders(ctx, &ordersProtocol.GetOrdersRequest{
		Id: request.Id,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to get orders")
	}

	return &protocol.GetProductOrdersResponse{
		Orders: resp.Orders,
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

	ordersCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dialOptions := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
	}
	if os.Getenv("ORDERS_GRPC_INSECURE") != "true" {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	} else {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}

	conn, err := grpc.DialContext(
		ordersCtx,
		os.Getenv("ORDERS_GRPC_URL"),
		dialOptions...,
	)
	if err != nil {
		fmt.Printf("unable to initialize orders client: %+v\n", err)
	}

	ordersClient = ordersProtocol.NewOrdersClient(conn)

	span.End()

	fmt.Printf("span end\n")

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "50051"
	}

	// start gRPC server
	var lis net.Listener
	lis, err = net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}

	grcpServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
	)
	reflection.Register(grcpServer)

	protocol.RegisterProductsServer(grcpServer, &productsServer{})
	log.Printf("server listening at %v", lis.Addr())
	if err := grcpServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
