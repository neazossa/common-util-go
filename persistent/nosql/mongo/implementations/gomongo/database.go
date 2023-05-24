package gomongo

import (
	"context"

	"github.com/neazzosa/common-util-go/logger/logger"
	"github.com/neazzosa/common-util-go/monitor/monitor"
	"github.com/neazzosa/common-util-go/persistent/nosql/mongo/mongo"
	"go.mongodb.org/mongo-driver/bson"
	mgo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	databaseImplementation struct {
		database *mgo.Database

		isMonitor      bool
		isCaptureError bool
		monitor        monitor.Monitor
		logger         logger.Logger
		requestID      string
		ctx            context.Context
	}
)

/*
===================================
	SET UP SECTION
===================================
*/

func NewDatabase(database *mgo.Database, logger logger.Logger) mongo.Database {
	return &databaseImplementation{database: database, logger: logger}
}
func (d *databaseImplementation) Monitor(ctx context.Context, mntr monitor.Monitor, requestID string, captureError bool) mongo.Database {
	return &databaseImplementation{
		database:       d.database,
		isMonitor:      true,
		isCaptureError: captureError,
		monitor:        mntr,
		logger:         d.logger,
		requestID:      requestID,
		ctx:            ctx,
	}
}

/*
===================================
	QUERY SECTION
===================================
*/

func (d *databaseImplementation) Drop(ctx context.Context) error {
	defer d.doMonitor("Drop")()
	return d.database.Drop(ctx)
}

func (d *databaseImplementation) CreateCollection(ctx context.Context, name string) error {
	defer d.doMonitor("CreateCollection")()
	return d.database.CreateCollection(ctx, name)
}

func (d *databaseImplementation) Collection(name string, opts ...*options.CollectionOptions) mongo.Collection {
	defer d.doMonitor("Collection")()
	if d.isMonitor {
		return NewCollection(d.database.Collection(name, opts...)).Monitor(d.ctx, d.monitor, d.logger, d.requestID, d.isCaptureError)
	}
	return NewCollection(d.database.Collection(name, opts...))
}

func (d *databaseImplementation) HasCollection(ctx context.Context, name string) bool {
	defer d.doMonitor("HasCollection")()
	collections, err := d.CollectionNames(ctx)
	if err != nil {
		return false
	}

	if len(collections) < 1 {
		return false
	}

	for _, collection := range collections {
		if collection == name {
			return true
		}
	}
	return false
}

func (d *databaseImplementation) CollectionNames(ctx context.Context) ([]string, error) {
	defer d.doMonitor("CollectionNames")()
	return d.database.ListCollectionNames(ctx, bson.D{})
}

func (d *databaseImplementation) doMonitor(action string) func() {
	if d.isMonitor {
		tr := d.startMonitor(action, d.database.Name())
		return func() {
			d.finishMonitor(tr)
		}
	}
	return func() {}
}

func (d *databaseImplementation) finishMonitor(transaction monitor.Transaction) {
	transaction.Finish()
}

func (d *databaseImplementation) startMonitor(action string, database string) monitor.Transaction {
	tags := []monitor.Tag{
		{"requestId", d.requestID},
		{"action", action},
		{"database", database},
	}

	return d.monitor.NewTransactionFromContext(d.ctx, monitor.Tick{
		Operation:       "db",
		TransactionName: action,
		Tags:            tags,
	})
}

func (d *databaseImplementation) captureError(err error) error {
	if d.isCaptureError {
		d.monitor.Capture(err)
	}
	return err
}
