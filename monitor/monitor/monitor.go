package monitor

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
)

type (
	Monitor interface {
		Capture(err error) *string
		CaptureMessage(msg string) *string
		SetScope(scope interface{}) Monitor

		Flush() bool
		Recover() *string

		//Transaction Tracking
		StartTransaction(ctx context.Context, span Tick) Transaction
		NewTransactionFromContext(ctx context.Context, tick Tick) Transaction

		//Echo Related
		EchoMiddleware(echoOption EchoOption) echo.MiddlewareFunc
		SetNewTransaction(next echo.HandlerFunc) echo.HandlerFunc
		CreateNewContextWithTransaction(e echo.Context, ctx context.Context) context.Context

		//GRPC
		GRPCServerMonitor() grpc.UnaryServerInterceptor
		GRPCClientMonitor() grpc.UnaryClientInterceptor
	}

	Transaction interface {
		StartChildTransaction(tick Tick) Transaction
		CreateNewTransactionContext(ctx context.Context) context.Context
		Finish()
		FinishWithTags(tags []Tag)
		Info() TransactionInfo
	}

	Tag struct {
		Key   string
		Value string
	}

	Tick struct {
		Operation       string
		TransactionName string
		Tags            []Tag
	}

	TransactionInfo struct {
		Tick
		Ctx        context.Context
		Start      time.Time
		End        time.Time
		Status     string
		StatusCode uint8
	}

	EchoOption struct {
		Repanic         bool
		WaitForDelivery bool
		Timeout         time.Duration
	}
)
