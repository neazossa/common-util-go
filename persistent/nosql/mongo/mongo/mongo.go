package mongo

import (
	"context"

	"github.com/neazossa/common-util-go/monitor/monitor"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mgo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	FindCallback func(Cursor, error) error

	Mongo interface {
		SetCollection(collection CollectionModel) Mongo
		Monitor(ctx context.Context, mntr monitor.Monitor, requestID string, captureError bool) Mongo
		Ping(ctx context.Context) error
		Close(ctx context.Context) error

		FindOne(ctx context.Context, filter, object interface{}, options ...*options.FindOneOptions) error
		FindAll(ctx context.Context, filter interface{}, results interface{}, options ...*options.FindOptions) error
		Find(ctx context.Context, filter interface{}, callback FindCallback, options ...*options.FindOptions) error

		FindOneAndDelete(ctx context.Context, filter interface{}, options ...*options.FindOneAndDeleteOptions) error
		FindOneAndUpdate(ctx context.Context, filter, object interface{}, options ...*options.FindOneAndUpdateOptions) error

		Insert(ctx context.Context, object interface{}, options ...*options.InsertOneOptions) (*primitive.ObjectID, error)
		InsertMany(ctx context.Context, documents []interface{}, options ...*options.InsertManyOptions) ([]primitive.ObjectID, error)

		Update(ctx context.Context, filter, object interface{}, options ...*options.UpdateOptions) error
		UpdateMany(ctx context.Context, filter, object interface{}, options ...*options.UpdateOptions) error

		DeleteMany(ctx context.Context, filter interface{}, options ...*options.DeleteOptions) error
		Delete(ctx context.Context, filter interface{}, options ...*options.DeleteOptions) error

		BulkDocument(ctx context.Context, data []mgo.WriteModel) error

		Count(ctx context.Context, opts ...*options.CountOptions) (int64, error)
		CountWithFilter(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error)

		Distinct(ctx context.Context, field string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error)
		Aggregate(ctx context.Context, pipeline interface{}, callback FindCallback, options ...*options.AggregateOptions) error

		DB() Database
	}
)
