# README #

How to use monitoring.

## Init Monitoring ##

Use preferred package for monitoring, then use the `New` function to initialize. Example for sentry:
```
sentry.NewSentryMonitoring(logger, option)
```

Also use preferred logger package that implement logger.Logger interface function

For NewSentryMonitoring, there is an option struct we can use : 
```
Dsn              string //required to start sentry
Debug            bool
AttachStacktrace bool //use true to attach stack trace when capturing error
IgnoreErrors     []string //regex
ServerName       string //required
Release          string
Dist             string
Environment      string
MaxBreadcrumbs   int
HTTPClient       *http.Client
HTTPTransport    http.RoundTripper
HTTPProxy        string
HTTPSProxy       string
FlushTimeout     time.Duration //required
SampleRate       float64 //from 0.0 - 1.0
TracesSampleRate float64 //from 0.0 - 1.0
```

## Capture Error ##

Use the `Capture(err error)` function to catch and send an error as an event to sentry. Example : 
```
monit := sentry.NewSentryMonitoring(logger, option)

err := errors.New("something went wrong")

monit.Capture(err)
```

## Capture With Scope ##

When we need to capture error with scope then use `SetScope(scope interface{})` before capture error. Make sure the scope using `sentry.Scope` in order to work with sentry. Example:

```
monit := sentry.NewSentryMonitoring(logger, option)

err := errors.New("something went wrong")

monit.SetScope(
    sentry.Scope{
        Tags: []sentry.Tag{
			{
				Key:   "someKey",
				Value: "someValue",
			},
			{
				Key:   "requestId",
				Value: "the-fake-request-id",
			},
			{
				Key:   "operation",
				Value: "testingservice.TestImpl()",
			},
		},
		Level: "info",
		User: sentry.User{
			ID:    "123",
			Email: "test@example.com",
		},
    }).Capture(err)
```

## Segment / Transaction Monitoring ##
For monitoring some specific segment or transaction, we can use `StartTransaction(ctx context.Context, span Tick)` than will start the sentry transaction and return `Transaction` interface. Example:
```
monit := sentry.NewSentryMonitoring(logger, option)

segment := monit.StartTransaction(context.Background(), Tick{
    Operation:          "service"
    TransactionName:    "someFunctionName"
    Tags:               []Tag{
        {"requestId", "the-fake-request-id"},
    }
})

//do some magic here

segment.Finish()

```

### Finish With Tag ###
We can also add more tags in the end of segment. Example:
```
monit := sentry.NewSentryMonitoring(logger, option)

segment := monit.StartTransaction(context.Background(), Tick{
    Operation:          "http"
    TransactionName:    "POST /ms-something/find-something"
    Tags:               []Tag{
        {"requestId", "the-fake-request-id"},
    }
})

//do some magic here

segment.FinishWithTags([]Tag{
        {"message", "success"},
        {"statusCode", 200},
    })

```

### Sub Segment / Transaction ###
We can also add sub-segment in current transaction. Please be noted that transaction name is unused in sub-segment. Please use operation instead. Example:
```
monit := sentry.NewSentryMonitoring(logger, option)

segment := monit.StartTransaction(context.Background(), Tick{
    Operation:          "http"
    TransactionName:    "POST /ms-something/find-something"
    Tags:               []Tag{
        {"requestId", "the-fake-request-id"},
    }
})

subSegment1 := segment.StartChildTransaction(Tick{
    Operation:          "redis"
    Tags:               []Tag{
        {"requestId", "the-fake-request-id"},
        {"action", "GET"},
    }
})
//do some magic here
subSegment1.Finish() 

subSegment2 := segment.StartChildTransaction(Tick{
    Operation:          "db"
    Tags:               []Tag{
        {"requestId", "the-fake-request-id"},
        {"action", "Find One"},
    }
})
//do some magic here
subSegment2.Finish() 

segment.FinishWithTags([]Tag{
        {"message", "success"},
        {"statusCode", 200},
    })

```

## Other Package Implementation ##
## Echo ##
Use this middleware to initialize, handling recover, and generate transaction / segment every echo http call.
```
e := echo.New()
monit := sentry.NewSentryMonitoring(logger, option)

e.Use(monit.EchoMiddleware(monitor.EchoOption{
    Repanic:         true,
    WaitForDelivery: true,
    Timeout:         500 * time.Millisecond,
}))
e.Use(monit.SetNewTransaction)
```

## GRPC ##
Use this function to create new transaction every time grpc was called or used.

### Client Side ###
```
monit := sentry.NewSentryMonitoring(logger, option)

var conn *grpc.ClientConn

conn, err := grpc.Dial("http://localhost:9000", grpc.WithInsecure(), grpc.WithUnaryInterceptor(monit.GRPCClientMonitor()))
...
```

### Server Side ###
```
monit := sentry.NewSentryMonitoring(logger, option)

grpcServer := grpc.NewServer(grpc.UnaryInterceptor(
    grpc_middleware.ChainUnaryServer(monit.GRPCServerMonitor())))
...
```

## Redis ##
When using monitor in redis, you just can use `Monitor(ctx context.Context, monitor monitor.Monitor, requestId string, captureError bool)` function. 

Use `captureBool = true` to automatically capturing error. Example:
```
monit := sentry.NewSentryMonitoring(logger, option)
cache, _ := NewRedisConnection(option, logger)

cache.
    Monitor(context.Background(), monit, "the-fake-request-id", true).
    Get("key", &result)
```

The monitor will capture transaction / segment as sub-segment if context have transaction or else will create new transaction / segment.

## Database ##
When using monitor in database, you just can use `Monitor(ctx context.Context, monitor monitor.Monitor, requestId string, captureError bool)` function.

Use `captureBool = true` to automatically capturing error. Example:
```
monit := sentry.NewSentryMonitoring(logger, option)
persist, _ := NewPostgresConnection(connection, logger)

persist.Monitor(context.Background(), monit, "the-fake-request-id", true).All(&result)
```

The monitor will capture transaction / segment as sub-segment if context have transaction or else will create new transaction / segment for every executed query.

## HTTP ##
When using monitor in database, you just can use `Monitor(ctx context.Context, monitor monitor.Monitor, requestId string, captureError bool)` function.

Use `captureBool = true` to automatically capturing error. Example:
```
monit := sentry.NewSentryMonitoring(logger, option)
persist := NewHttpRestyClient(contex.Background(), logger)

response, err := persist.Monitor(context.Background(), monit, "the-fake-request-id", true).
    GET("path", map[string]string{}, nil)
```

The monitor will capture transaction / segment as sub-segment if context have transaction or else will create new transaction / segment for every rest call.