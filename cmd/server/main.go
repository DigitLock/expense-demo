package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	expensev1 "github.com/digitlock/expense-demo/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type expenseServer struct {
	expensev1.UnimplementedExpenseServiceServer
	expenses []expensev1.Expense
}

func (s *expenseServer) AddExpense(ctx context.Context, req *expensev1.AddExpenseRequest) (*expensev1.AddExpenseResponse, error) {
	// Простая валидация
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than 0")
	}
	if req.Category == "" {
		return nil, status.Error(codes.InvalidArgument, "category is required")
	}

	exp := expensev1.Expense{
		Id:         fmt.Sprintf("%d", len(s.expenses)+1),
		Name:       req.Name,
		Amount:     req.Amount,
		Category:   req.Category,
		Note:       req.Note,
		OccurredAt: req.OccurredAt,
		CreatedAt:  "2025-10-16T12:00:00Z",
	}
	s.expenses = append(s.expenses, exp)

	return &expensev1.AddExpenseResponse{
		Id:     exp.Id,
		Status: "ok",
	}, nil
}

func (s *expenseServer) ListExpenses(ctx context.Context, req *expensev1.ListExpensesRequest) (*expensev1.ListExpensesResponse, error) {
	var expenses []*expensev1.Expense
	for i := range s.expenses {
		expenses = append(expenses, &s.expenses[i])
	}
	return &expensev1.ListExpensesResponse{Expenses: expenses}, nil
}

func (s *expenseServer) GetSummary(ctx context.Context, req *expensev1.SummaryRequest) (*expensev1.SummaryResponse, error) {
	summary := make(map[string]*expensev1.CategorySummary)

	for _, e := range s.expenses {
		if _, ok := summary[e.Category]; !ok {
			summary[e.Category] = &expensev1.CategorySummary{
				Category: e.Category,
				Items:    0,
				Total:    0,
			}
		}
		summary[e.Category].Items++
		summary[e.Category].Total += e.Amount
	}

	var result []*expensev1.CategorySummary
	for _, cat := range summary {
		result = append(result, cat)
	}

	return &expensev1.SummaryResponse{Summaries: result}, nil
}

func main() {
	grpcPort := flag.Int("grpc-port", 8091, "gRPC server port")
	httpPort := flag.Int("http-port", 8092, "HTTP gateway port")
	flag.Parse()

	s := &expenseServer{}

	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		grpcServer := grpc.NewServer()
		expensev1.RegisterExpenseServiceServer(grpcServer, s)
		reflection.Register(grpcServer)
		log.Printf("gRPC server running on :%d", *grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	ctx := context.Background()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := expensev1.RegisterExpenseServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", *grpcPort), opts)
	if err != nil {
		log.Fatalf("failed to register handler: %v", err)
	}
	log.Printf("REST gateway running on :%d", *httpPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), mux); err != nil {
		log.Fatalf("failed to serve http: %v", err)
	}
}
