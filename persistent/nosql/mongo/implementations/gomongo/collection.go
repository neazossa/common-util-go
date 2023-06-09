package gomongo

import (
	"context"

	"github.com/neazossa/common-util-go/logger/logger"
	"github.com/neazossa/common-util-go/monitor/monitor"
	"github.com/neazossa/common-util-go/persistent/nosql/mongo/mongo"
	mgo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	collectionImplementation struct {
		collection *mgo.Collection

		isMonitor      bool
		isCaptureError bool
		monitor        monitor.Monitor
		logger         logger.Logger
		requestID      string
		ctx            context.Context
	}
)

func NewCollection(collection *mgo.Collection) mongo.Collection {
	coll := collectionImplementation{collection: collection}
	return &coll
}

func (c *collectionImplementation) Monitor(ctx context.Context, mntr monitor.Monitor, log logger.Logger, requestID string, captureError bool) mongo.Collection {
	return &collectionImplementation{
		collection:     c.collection,
		isMonitor:      true,
		isCaptureError: captureError,
		monitor:        mntr,
		logger:         log,
		requestID:      requestID,
		ctx:            ctx,
	}
}

/*
===================================
	QUERY SECTION
===================================
*/

func (c *collectionImplementation) DropCollection(ctx context.Context) error {
	return c.collection.Drop(ctx)
}

func (c *collectionImplementation) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mgo.Cursor, error) {
	return c.collection.Aggregate(ctx, pipeline, opts...)
}

func (c *collectionImplementation) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mgo.Cursor, error) {
	return c.collection.Find(ctx, filter, opts...)
}

func (c *collectionImplementation) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mgo.SingleResult {
	return c.collection.FindOne(ctx, filter, opts...)
}

func (c *collectionImplementation) BulkWrite(ctx context.Context, models []mgo.WriteModel, opts ...*options.BulkWriteOptions) (*mgo.BulkWriteResult, error) {
	return c.collection.BulkWrite(ctx, models, opts...)
}

func (c *collectionImplementation) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return c.collection.CountDocuments(ctx, filter, opts...)
}

func (c *collectionImplementation) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mgo.DeleteResult, error) {
	return c.collection.DeleteOne(ctx, filter, opts...)
}

func (c *collectionImplementation) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mgo.DeleteResult, error) {
	return c.collection.DeleteMany(ctx, filter, opts...)
}

func (c *collectionImplementation) UpdateMany(ctx context.Context, filter, update interface{}, opts ...*options.UpdateOptions) (*mgo.UpdateResult, error) {
	return c.collection.UpdateMany(ctx, filter, update, opts...)
}

func (c *collectionImplementation) UpdateOne(ctx context.Context, filter, update interface{}, opts ...*options.UpdateOptions) (*mgo.UpdateResult, error) {
	return c.collection.UpdateOne(ctx, filter, update, opts...)
}

func (c *collectionImplementation) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mgo.InsertManyResult, error) {
	return c.collection.InsertMany(ctx, documents, opts...)
}

func (c *collectionImplementation) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mgo.InsertOneResult, error) {
	return c.collection.InsertOne(ctx, document, opts...)
}

func (c *collectionImplementation) FindOneAndUpdate(ctx context.Context, filter, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mgo.SingleResult {
	return c.collection.FindOneAndUpdate(ctx, filter, update, opts...)
}

func (c *collectionImplementation) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mgo.SingleResult {
	return c.collection.FindOneAndDelete(ctx, filter, opts...)
}

func (c *collectionImplementation) Distinct(ctx context.Context, field string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	return c.collection.Distinct(ctx, field, filter, opts...)
}
