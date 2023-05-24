package gomongo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/neazossa/common-util-go/logger/logger"
	"github.com/neazossa/common-util-go/monitor/monitor"
	"github.com/neazossa/common-util-go/persistent/nosql/mongo/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mgo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type (
	implementation struct {
		client         *mgo.Client
		database       mongo.Database
		databaseName   string
		collectionName string
		collection     mongo.Collection

		isMonitor      bool
		isCaptureError bool
		monitor        monitor.Monitor
		logger         logger.Logger
		requestID      string
		ctx            context.Context
	}
	decoder struct {
	}
	Connection struct {
		Host     string
		Username string
		Password string
		Database string
	}
)

/*
===================================
	SET UP SECTION
===================================
*/

func (d decoder) DecodeValue(dCtx bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Kind() != reflect.String {
		return fmt.Errorf("bad type or not settable")
	}
	var str string
	var err error
	switch vr.Type() {
	case bsontype.String:
		if str, err = vr.ReadString(); err != nil {
			return err
		}
	case bsontype.Null: // THIS IS THE MISSING PIECE TO HANDLE NULL!
		if err = vr.ReadNull(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot decode %v into a string type", vr.Type())
	}

	val.SetString(str)
	return nil
}

func NewMongoConnection(ctx context.Context, connection Connection, logger logger.Logger) (mongo.Mongo, error) {

	clientOptions := options.Client()

	usernamePassword := ""
	if connection.Password != "" || connection.Username != "" {
		if connection.Username == "" {
			return nil, errors.New("username cannot empty")
		}
		if connection.Password == "" {
			return nil, errors.New("password cannot empty")
		}
		usernamePassword = fmt.Sprintf("%s:%s@", connection.Username, connection.Password)
	}

	url := fmt.Sprintf("mongodb://%s%s/%s", usernamePassword, connection.Host, connection.Database)

	clientOptions.ApplyURI(url).
		SetRegistry(bson.NewRegistryBuilder().RegisterTypeDecoder(reflect.TypeOf(""), decoder{}).Build())

	client, err := mgo.Connect(ctx, clientOptions)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %s", err.Error())
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping mongo : %s", err.Error())
	}

	database := NewDatabase(client.Database(connection.Database), logger)

	return &implementation{
		client:       client,
		database:     database,
		logger:       logger,
		databaseName: connection.Database,
	}, nil
}

func (i *implementation) SetCollection(collection mongo.CollectionModel) mongo.Mongo {
	return &implementation{
		client:         i.client,
		database:       i.database,
		logger:         i.logger,
		databaseName:   i.databaseName,
		collectionName: collection.CollectionName(),
		collection:     i.database.Collection(collection.CollectionName()),
	}
}

func (i *implementation) Monitor(ctx context.Context, mntr monitor.Monitor, requestID string, captureError bool) mongo.Mongo {
	return &implementation{
		client:         i.client,
		database:       i.database.Monitor(ctx, mntr, requestID, captureError),
		databaseName:   i.databaseName,
		collectionName: i.collectionName,
		collection:     i.collection,
		isMonitor:      true,
		isCaptureError: captureError,
		monitor:        mntr,
		logger:         i.logger,
		requestID:      requestID,
		ctx:            ctx,
	}
}

func (i *implementation) Ping(ctx context.Context) error {
	ctxData, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return i.client.Ping(ctxData, readpref.Primary())
}

func (i *implementation) Client() *mgo.Client {
	return i.client
}

func (i *implementation) Close(ctx context.Context) error {
	return i.client.Disconnect(ctx)
}

func (i *implementation) DB() mongo.Database {
	return i.database
}

/*
===================================
	QUERY SECTION
===================================
*/

func (i *implementation) FindAll(ctx context.Context, filter interface{}, results interface{}, options ...*options.FindOptions) error {
	defer i.doMonitor("FindAll")()
	rs, err := i.collection.Find(ctx, filter, options...)

	if err != nil {
		return fmt.Errorf("failed to find all with context : %+v", err)
	}

	if err := rs.All(ctx, results); err != nil {
		return fmt.Errorf("failed to decode all : %+v", err)
	}

	return nil
}

func (i *implementation) FindOne(ctx context.Context, filter, object interface{}, options ...*options.FindOneOptions) error {
	defer i.doMonitor("FindOne")()
	sr := i.collection.FindOne(ctx, filter, options...)

	if err := sr.Err(); err != nil {
		return fmt.Errorf("FindOne failed: %+v", err)
	}

	if err := sr.Decode(object); err != nil {
		return fmt.Errorf("FindOne decode failed: %+v", err)
	}

	return nil
}

func (i *implementation) Find(ctx context.Context, filter interface{}, callback mongo.FindCallback, options ...*options.FindOptions) error {
	defer i.doMonitor("Find")()
	coll, err := i.collection.Find(ctx, filter, options...)
	if err != nil {
		return err
	}

	cursor, err := NewCursor(coll)

	defer func() {
		if cursor == nil {
			return
		}

		if err := cursor.Close(ctx); err != nil {
			i.logger.Errorf("failed to close cursor %s", err.Error())
		}
	}()

	if err != nil {
		return callback(nil, err)
	} else {
		return callback(cursor, nil)
	}
}

func (i *implementation) FindOneAndDelete(ctx context.Context, filter interface{}, options ...*options.FindOneAndDeleteOptions) error {
	defer i.doMonitor("FindOneAndDelete")()
	sr := i.collection.FindOneAndDelete(ctx, filter, options...)

	if err := sr.Err(); err != nil {
		return fmt.Errorf("FindOneAndDeleteWithContext failed: %+v", err)
	}

	return nil
}

func (i *implementation) FindOneAndUpdate(ctx context.Context, filter, object interface{}, options ...*options.FindOneAndUpdateOptions) error {
	defer i.doMonitor("FindOneAndUpdate")()
	sr := i.collection.FindOneAndUpdate(ctx, filter, object, options...)

	if err := sr.Err(); err != nil {
		return fmt.Errorf("FindOneAndUpdateWithContext failed: %+v", err)
	}

	if err := sr.Decode(&object); err != nil {
		return fmt.Errorf("FindOneAndUpdate decode failed: %+v", err)
	}

	return nil
}

func (i *implementation) Insert(ctx context.Context, object interface{}, options ...*options.InsertOneOptions) (*primitive.ObjectID, error) {
	defer i.doMonitor("Insert")()
	ir, err := i.collection.InsertOne(ctx, object, options...)

	if err != nil {
		return nil, fmt.Errorf("InsertOneWithContext failed: %+v", err)
	}

	id, ok := ir.InsertedID.(primitive.ObjectID)

	if !ok {
		return nil, fmt.Errorf("InsertWithContext failed to cast ObjectID")
	}

	return &id, nil
}

func (i *implementation) InsertMany(ctx context.Context, documents []interface{}, options ...*options.InsertManyOptions) ([]primitive.ObjectID, error) {
	defer i.doMonitor("InsertMany")()
	ir, err := i.collection.InsertMany(ctx, documents, options...)

	if err != nil {
		return nil, fmt.Errorf("InsertManyWithContext failed: %+v", err)
	}

	ids := make([]primitive.ObjectID, 0)

	for _, id := range ir.InsertedIDs {
		i, ok := id.(primitive.ObjectID)

		if !ok {
			err = fmt.Errorf("InsertWithContext failed to cast ObjectID %s", i)
			break
		}

		ids = append(ids, i)
	}

	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (i *implementation) Update(ctx context.Context, filter, object interface{}, options ...*options.UpdateOptions) error {
	defer i.doMonitor("Update")()
	if _, err := i.collection.UpdateOne(ctx, filter, object, options...); err != nil {
		return fmt.Errorf("UpdateWithContext failed: %+v", err)
	}

	return nil
}

func (i *implementation) UpdateMany(ctx context.Context, filter, object interface{}, options ...*options.UpdateOptions) error {
	defer i.doMonitor("UpdateMany")()
	if _, err := i.collection.UpdateMany(ctx, filter, object, options...); err != nil {
		return fmt.Errorf("UpdateManyWithContext failed: %+v", err)
	}

	return nil
}

func (i *implementation) DeleteMany(ctx context.Context, filter interface{}, options ...*options.DeleteOptions) error {
	defer i.doMonitor("DeleteMany")()
	if _, err := i.collection.DeleteMany(ctx, filter, options...); err != nil {
		return fmt.Errorf("DeleteManyWithContext failed: %+v", err)
	}

	return nil
}

func (i *implementation) Delete(ctx context.Context, filter interface{}, options ...*options.DeleteOptions) error {
	defer i.doMonitor("Delete")()
	if _, err := i.collection.DeleteOne(ctx, filter, options...); err != nil {
		return fmt.Errorf("DeleteWithContext failed: %+v", err)
	}

	return nil
}

func (i *implementation) CountWithFilter(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	defer i.doMonitor("CountWithFilter")()
	return i.countWithFilter(ctx, filter, opts...)
}

func (i *implementation) Distinct(ctx context.Context, field string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	defer i.doMonitor("Distinct")()
	object, err := i.collection.Distinct(ctx, field, filter, opts...)
	if err != nil {
		return nil, fmt.Errorf("Distinct failed: %+v", err)
	}

	return object, nil
}

func (i *implementation) Count(ctx context.Context, opts ...*options.CountOptions) (int64, error) {
	defer i.doMonitor("Count")()
	return i.countWithFilter(ctx, bson.D{}, opts...)
}

func (i *implementation) BulkDocument(ctx context.Context, data []mgo.WriteModel) error {
	defer i.doMonitor("BulkDocument")()
	_, err := i.collection.BulkWrite(ctx, data)
	if err != nil {
		return err
	}
	return nil
}

func (i *implementation) Aggregate(ctx context.Context, pipeline interface{}, callback mongo.FindCallback, options ...*options.AggregateOptions) error {
	defer i.doMonitor("Aggregate")()
	coll, err := i.collection.Aggregate(ctx, pipeline, options...)
	if err != nil {
		return err
	}

	cursor, err := NewCursor(coll)

	defer func() {
		if cursor == nil {
			return
		}

		if err := cursor.Close(ctx); err != nil {
			i.logger.Errorf("failed to close cursor %s", err.Error())
		}
	}()

	if err != nil {
		return callback(nil, err)
	} else {
		return callback(cursor, nil)
	}
}

func (i *implementation) countWithFilter(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	coll := i.collection
	total, err := coll.CountDocuments(ctx, filter, opts...)

	if err != nil {
		return 0, fmt.Errorf("failed to count collection %s: %s", i.collectionName, err.Error())
	}

	return total, nil
}

func (i *implementation) doMonitor(action string) func() {
	if i.isMonitor {
		tr := i.startMonitor(action, i.databaseName, i.collectionName)
		return func() {
			i.finishMonitor(tr)
		}
	}
	return func() {}
}

func (i *implementation) finishMonitor(transaction monitor.Transaction) {
	transaction.Finish()
}

func (i *implementation) startMonitor(action string, database, table string) monitor.Transaction {
	tags := []monitor.Tag{
		{"requestId", i.requestID},
		{"action", action},
		{"database", database},
		{"collection", table},
	}

	return i.monitor.NewTransactionFromContext(i.ctx, monitor.Tick{
		Operation:       "mongo",
		TransactionName: action,
		Tags:            tags,
	})
}

func (i *implementation) captureError(err error) error {
	if i.isCaptureError {
		i.monitor.Capture(err)
	}
	return err
}
