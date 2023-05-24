package mongo

import (
	"context"

	"github.com/neazzosa/common-util-go/logger/logger"
	"github.com/neazzosa/common-util-go/monitor/monitor"
	mgo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	CollectionModel interface {
		CollectionName() string
	}

	Collection interface {
		Monitor(ctx context.Context, mntr monitor.Monitor, log logger.Logger, requestID string, captureError bool) Collection

		DropCollection(ctx context.Context) error

		Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mgo.Cursor, error)

		Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mgo.Cursor, error)
		FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mgo.SingleResult

		BulkWrite(ctx context.Context, models []mgo.WriteModel, opts ...*options.BulkWriteOptions) (*mgo.BulkWriteResult, error)
		CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error)

		DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mgo.DeleteResult, error)
		DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mgo.DeleteResult, error)

		UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mgo.UpdateResult, error)
		UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mgo.UpdateResult, error)

		InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mgo.InsertManyResult, error)
		InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mgo.InsertOneResult, error)

		FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mgo.SingleResult
		FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mgo.SingleResult

		Distinct(ctx context.Context, field string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error)
	}
)
