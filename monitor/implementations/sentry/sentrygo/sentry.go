package sentrygo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/neazzosa/common-util-go/logger/logger"
	"github.com/neazzosa/common-util-go/monitor/monitor"
	"github.com/neazzosa/common-util-go/shared/shared"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	GRPCServerLoggerFormat = "entry from: %s ; request: %v"
	GRPCClientLoggerFormat = "response from: %s ; request: %v ; response %v"
)

type (
	Option struct {
		Dsn              string
		Debug            bool
		AttachStacktrace bool
		IgnoreErrors     []string //regex
		ServerName       string
		Release          string
		Dist             string
		Environment      string
		MaxBreadcrumbs   int
		HTTPClient       *http.Client
		HTTPTransport    http.RoundTripper
		HTTPProxy        string
		HTTPSProxy       string
		FlushTimeout     time.Duration
		SampleRate       float64
		TracesSampleRate float64
	}

	User struct {
		ID    string
		Email string
	}

	Scope struct {
		Tags           []monitor.Tag
		Level          string
		User           User
		Breadcrumb     sentry.Breadcrumb
		BreadcrumbHint sentry.BreadcrumbHint
	}

	sentryMonitor struct {
		logger logger.Logger
		option Option
		hub    *sentry.Hub
		scope  *Scope
	}

	transaction struct {
		tick monitor.Tick
		span *sentry.Span
	}

	basicFunc func() *string
)

func NewSentryMonitoring(logger logger.Logger, option Option) (monitor.Monitor, error) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              option.Dsn,
		Debug:            option.Debug,
		AttachStacktrace: option.AttachStacktrace,
		ServerName:       option.ServerName,
		Release:          option.Release,
		Dist:             option.Dist,
		Environment:      option.Environment,
		MaxBreadcrumbs:   option.MaxBreadcrumbs,
		HTTPClient:       option.HTTPClient,
		HTTPTransport:    option.HTTPTransport,
		HTTPProxy:        option.HTTPProxy,
		HTTPSProxy:       option.HTTPSProxy,
		IgnoreErrors:     option.IgnoreErrors,
		SampleRate:       option.SampleRate,
		TracesSampleRate: option.TracesSampleRate,
	})

	if err != nil {
		logger.Fatalf("failed to start sentry monitoring", err)
		return nil, err
	}

	return &sentryMonitor{
		logger: logger,
		option: option,
		hub:    sentry.CurrentHub(),
	}, nil
}

func (s *sentryMonitor) Capture(err error) *string {
	if err == nil {
		return nil
	}

	if s.scope != nil {
		return s.implementScope(func() *string {
			return getEventId(s.cloneHub().CaptureException(err))
		})
	}
	return getEventId(s.cloneHub().CaptureException(err))
}

func (s *sentryMonitor) CaptureMessage(msg string) *string {
	if s.scope != nil {
		return s.implementScope(func() *string {
			return getEventId(s.cloneHub().CaptureMessage(msg))
		})
	}
	return getEventId(s.cloneHub().CaptureMessage(msg))
}

func (s *sentryMonitor) Flush() bool {
	return s.hub.Flush(s.option.FlushTimeout)
}

func (s *sentryMonitor) SetScope(scope interface{}) monitor.Monitor {
	sc, ok := scope.(Scope)
	if !ok {
		s.logger.Fatal("failed to set sentry scope")
		return s
	}

	return &sentryMonitor{
		logger: s.logger,
		option: s.option,
		hub:    s.hub,
		scope:  &sc,
	}
}

func (s *sentryMonitor) Recover() *string {
	var (
		err error
	)

	r := recover()

	switch r.(type) {
	case string:
		err = errors.New(r.(string))
	case error:
		err = r.(error)
	default:
		err = errors.New(fmt.Sprintf("unknown error : %v", r))
	}
	return s.Capture(err)
}

func (s *sentryMonitor) StartTransaction(ctx context.Context, tick monitor.Tick) monitor.Transaction {
	var (
		sp *sentry.Span
	)

	if len(tick.TransactionName) == 0 {
		sp = sentry.StartSpan(ctx, tick.Operation)
	} else {
		sp = sentry.StartSpan(ctx, tick.Operation, sentry.TransactionName(tick.TransactionName))
	}

	for _, tag := range tick.Tags {
		sp.SetTag(tag.Key, tag.Value)
	}

	return &transaction{
		tick: tick,
		span: sp,
	}
}

func (s *sentryMonitor) NewTransactionFromContext(ctx context.Context, tick monitor.Tick) monitor.Transaction {
	if val := ctx.Value("transaction"); val != nil {
		if ctxTr, ok := val.(monitor.Transaction); ok {
			return ctxTr.StartChildTransaction(monitor.Tick{
				Operation: tick.Operation,
				Tags:      tick.Tags,
			})
		}
	}
	return s.StartTransaction(ctx, tick)
}

func (s *sentryMonitor) EchoMiddleware(echoOption monitor.EchoOption) echo.MiddlewareFunc {
	return sentryecho.New(sentryecho.Options{
		Repanic:         echoOption.Repanic,
		WaitForDelivery: echoOption.WaitForDelivery,
		Timeout:         echoOption.Timeout,
	})
}

func (s *sentryMonitor) SetNewTransaction(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		if strings.Contains(ctx.Request().URL.Path, "swagger") {
			return next(ctx)
		}

		if hub := sentryecho.GetHubFromContext(ctx); hub != nil {
			s.hub = hub.Clone()

			requestId := ""
			if response := ctx.Get("Response"); response != nil {
				if mapData, ok := response.(map[string]interface{}); ok && mapData != nil {
					if data, ok := mapData["service_request_id"]; ok && data != nil {
						if requestUuid, ok := data.(*uuid.UUID); ok {
							requestId = requestUuid.String()
						}
					}
				}
			}

			tags := []monitor.Tag{
				{"requestID", requestId},
			}

			tr := s.StartTransaction(ctx.Request().Context(), monitor.Tick{
				Operation:       "echo",
				TransactionName: fmt.Sprintf("%s %s", ctx.Request().Method, ctx.Request().URL),
				Tags:            tags,
			})

			ctx.Set("transaction", tr)

			defer tr.FinishWithTags(tags)
		}
		return next(ctx)
	}
}

func (s *sentryMonitor) CreateNewContextWithTransaction(e echo.Context, ctx context.Context) context.Context {
	tr := e.Get("transaction")

	if tr != nil {
		if trans, ok := tr.(monitor.Transaction); ok {
			return trans.CreateNewTransactionContext(ctx)
		}
	}

	return ctx
}

func (s *sentryMonitor) GRPCServerMonitor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		request, _ := json.Marshal(req)
		m := make(map[string]interface{})
		json.Unmarshal(request, &m)
		s.logger.Info(fmt.Sprintf(GRPCServerLoggerFormat, info.FullMethod, m))

		requestId, _ := shared.GetRequestId(req)

		tr := s.StartTransaction(ctx, monitor.Tick{
			Operation:       "grpc.server",
			TransactionName: info.FullMethod,
			Tags: []monitor.Tag{
				{"requestId", requestId},
			},
		})

		ctx2 := tr.CreateNewTransactionContext(ctx)

		res, err := handler(ctx2, req)
		if err != nil {
			s.Capture(err)
			s.logger.Error(err)
		}

		defer func() {
			errStats, _ := status.FromError(err)
			tr.FinishWithTags([]monitor.Tag{
				{"code", fmt.Sprintf("%d", errStats.Code())},
				{"status", errStats.Code().String()},
				{"message", errStats.Message()},
			})
		}()
		return res, err
	}
}

func (s *sentryMonitor) GRPCClientMonitor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		requestId, _ := shared.GetRequestId(req)
		tr := s.NewTransactionFromContext(ctx, monitor.Tick{
			Operation:       "grpc.client",
			TransactionName: method,
			Tags: []monitor.Tag{
				{"requestId", requestId},
				{"action", method},
			},
		})

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			s.logger.Error(err)
			s.Capture(err)
		}
		defer func() {
			errStats, _ := status.FromError(err)
			tr.FinishWithTags([]monitor.Tag{
				{"code", fmt.Sprintf("%d", errStats.Code())},
				{"status", errStats.Code().String()},
				{"message", errStats.Message()},
			})
			request, _ := json.Marshal(req)
			response, _ := json.Marshal(reply)
			mReq := make(map[string]interface{})
			mRes := make(map[string]interface{})
			json.Unmarshal(request, &mReq)
			json.Unmarshal(response, &mRes)
			s.logger.Info(fmt.Sprintf(GRPCClientLoggerFormat, method, mReq, mRes))
		}()
		return err
	}
}

func (t *transaction) Finish() {
	t.span.Finish()
}

func (t *transaction) FinishWithTags(tags []monitor.Tag) {
	for _, tag := range tags {
		t.span.SetTag(tag.Key, tag.Value)
	}
	t.span.Finish()
}

func (t *transaction) Info() monitor.TransactionInfo {
	return monitor.TransactionInfo{
		Tick:       t.tick,
		Ctx:        t.span.Context(),
		Start:      t.span.StartTime,
		End:        t.span.EndTime,
		Status:     t.span.Status.String(),
		StatusCode: uint8(t.span.Status),
	}
}

func (t *transaction) StartChildTransaction(tick monitor.Tick) monitor.Transaction {
	sp := t.span.StartChild(tick.Operation)

	for _, tag := range tick.Tags {
		sp.SetTag(tag.Key, tag.Value)
	}

	return &transaction{
		tick: tick,
		span: sp,
	}
}

func (t *transaction) CreateNewTransactionContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "transaction", t)
}

func getEventId(e *sentry.EventID) *string {
	if e != nil {
		id := string(*e)
		return &id
	}
	return nil
}

func (s *sentryMonitor) cloneHub() *sentry.Hub {
	return s.hub.Clone()
}

func (s *sentryMonitor) implementScope(f basicFunc) *string {
	var (
		id *string
	)

	s.cloneHub().WithScope(func(scope *sentry.Scope) {
		switch strings.ToLower(s.scope.Level) {
		case string(sentry.LevelDebug):
			scope.SetLevel(sentry.LevelDebug)
		case string(sentry.LevelError):
			scope.SetLevel(sentry.LevelError)
		case string(sentry.LevelFatal):
			scope.SetLevel(sentry.LevelFatal)
		case string(sentry.LevelWarning):
			scope.SetLevel(sentry.LevelWarning)
		default:
			scope.SetLevel(sentry.LevelInfo)
		}

		for _, tag := range s.scope.Tags {
			scope.SetTag(tag.Key, tag.Value)
		}
		scope.SetUser(sentry.User{
			Email: s.scope.User.Email,
			ID:    s.scope.User.ID,
		})
		id = f()
	})
	return id
}
