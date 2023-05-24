package mongo

import (
	"context"

	"github.com/neazzosa/common-util-go/monitor/monitor"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	Database interface {
		Collection(name string, opts ...*options.CollectionOptions) Collection
		Drop(ctx context.Context) error
		CreateCollection(ctx context.Context, name string) error
		HasCollection(ctx context.Context, name string) bool
		CollectionNames(ctx context.Context) ([]string, error)

		Monitor(ctx context.Context, mntr monitor.Monitor, requestID string, captureError bool) Database
	}
)
