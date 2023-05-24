package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/neazzosa/common-util-go/persistent/sql/sql"
	"gorm.io/gorm/clause"

	"github.com/neazzosa/common-util-go/logger/logger"
	"github.com/neazzosa/common-util-go/monitor/monitor"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type (
	SQL struct {
		Logger         logger.Logger
		db             *gorm.DB
		DryRun         bool
		connection     sql.Connection
		isMonitor      bool
		monitor        monitor.Monitor
		ctx            context.Context
		isCaptureError bool
		requestId      string
	}
)

func NewPostgresConnection(connection sql.Connection, logger logger.Logger) (sql.ORM, error) {

	dsn := "host=" + connection.Host +
		" user=" + connection.Username +
		" password=" + connection.Password +
		" dbname=" + connection.DBName +
		" port=" + connection.Port +
		" sslmode=" + connection.SSLMode +
		" TimeZone=" + connection.Timezone

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Panicln(err)
		panic(err)
	}

	persist := SQL{
		Logger:     logger,
		DryRun:     connection.SQLDebug,
		connection: connection,
	}
	persist.SetDB(db)
	return &persist, err
}

func (s *SQL) SetDB(db *gorm.DB) {
	s.db = db
}

func (s *SQL) Where(query interface{}, args ...interface{}) sql.ORM {
	db := s.db.Where(query, args)
	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) OrWhere(query interface{}, args ...interface{}) sql.ORM {
	db := s.db.Or(query, args)
	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) First(result interface{}) error {
	defer s.doMonitor("GET FIRST", s.connection.DBName, result.(schema.Tabler).TableName())()
	db := s.db.First(result)

	if err := db.Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.captureError(errors.Wrap(err, "error fetch data from database"))
		} else {
			return s.captureError(err)
		}
	}

	return nil
}

func (s *SQL) All(result interface{}) (int64, error) {
	defer s.doMonitor("GET ALL", s.connection.DBName, result.(schema.Tabler).TableName())()
	if s.DryRun {
		db := s.db.Session(&gorm.Session{DryRun: s.DryRun}).Find(result)
		s.Logger.Debug(db.Statement.Explain(db.Statement.SQL.String(), db.Statement.Vars...))
	}

	db := s.db.Find(result)
	if err := db.Error; err != nil {
		return int64(0), s.captureError(errors.Wrap(err, "error fetch data from database"))
	}

	return db.RowsAffected, nil
}

func (s *SQL) Order(value interface{}) sql.ORM {
	db := s.db.Order(value)
	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Limit(value int) sql.ORM {
	db := s.db.Limit(value)
	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Offset(value int) sql.ORM {
	db := s.db.Offset(value)
	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Create(data interface{}) error {
	defer s.doMonitor("CREATE", s.connection.DBName, getTableName(data))()
	if s.DryRun {
		db := s.db.Session(&gorm.Session{DryRun: s.DryRun}).Create(data)
		s.Logger.Debug(db.Statement.Explain(db.Statement.SQL.String(), db.Statement.Vars...))
	}

	return s.captureError(s.db.Create(data).Error)
}

// Update : do update all given data
// Use Patch() to partial update
func (s *SQL) Update(data interface{}) error {
	defer s.doMonitor("UPDATE", s.connection.DBName, getTableName(data))()
	if s.DryRun {
		db := s.db.Session(&gorm.Session{DryRun: true}).Save(data)
		s.Logger.Debug(db.Statement.Explain(db.Statement.SQL.String(), db.Statement.Vars...))
	}

	return s.captureError(s.db.Save(data).Error)
}

// Patch : only update non-zero fields
// Use Update() to update all data
func (s *SQL) Patch(data interface{}, fields ...string) error {
	defer s.doMonitor("PATCH", s.connection.DBName, getTableName(data))()
	db := s.db
	if s.DryRun {
		dbDry := db.Session(&gorm.Session{DryRun: true})
		if len(fields) > 0 {
			dbDry = dbDry.Select(fields)
		}
		dbDry.Updates(data)
		s.Logger.Debug(dbDry.Statement.Explain(dbDry.Statement.SQL.String(), dbDry.Statement.Vars...))
	}

	if len(fields) > 0 {
		db = db.Select(fields)
	}

	return s.captureError(db.Updates(data).Error)
}

func (s *SQL) Delete(data interface{}, cond ...interface{}) error {
	defer s.doMonitor("DELETE", s.connection.DBName, getTableName(data))()
	if s.DryRun {
		db := s.db.Session(&gorm.Session{DryRun: true}).Delete(data, cond)
		s.Logger.Debug(db.Statement.Explain(db.Statement.SQL.String(), db.Statement.Vars...))
	}

	return s.captureError(s.db.Delete(data, cond).Error)
}

// Upsert : can upsert many values at once
func (s *SQL) Upsert(chunkSize int, data interface{}, onConflict sql.OnConflict) error {
	defer s.doMonitor("UPSERT", s.connection.DBName, getTableName(data))()

	var (
		columns = make([]clause.Column, 0)
	)

	for _, col := range onConflict.UniqueColumn {
		columns = append(columns, clause.Column{
			Name: col,
		})
	}

	if s.DryRun {
		db := s.db.Session(&gorm.Session{DryRun: true}).Clauses(clause.OnConflict{
			Columns:   columns,
			DoUpdates: clause.AssignmentColumns(onConflict.OnlyUpdate),
		})
		if chunkSize > 0 {
			db = db.CreateInBatches(data, chunkSize)
		} else {
			db = db.Create(data)
		}
		s.Logger.Debug(db.Statement.Explain(db.Statement.SQL.String(), db.Statement.Vars...))
	}

	db := s.db.Clauses(clause.OnConflict{
		Columns:   columns,
		DoUpdates: clause.AssignmentColumns(onConflict.OnlyUpdate),
	}).Create(data)

	if chunkSize > 0 {
		db = db.CreateInBatches(data, chunkSize)
	} else {
		db = db.Create(data)
	}
	return s.captureError(db.Error)
}

func (s *SQL) Exec(sql string, args ...interface{}) error {
	defer s.doMonitor("EXEC RAW", s.connection.DBName, "")()
	if s.DryRun {
		db := s.db.Session(&gorm.Session{DryRun: true}).Exec(sql, args...)
		s.Logger.Debug(db.Statement.Explain(db.Statement.SQL.String(), db.Statement.Vars...))
	}

	res := s.db.Exec(sql, args...)

	if err := res.Error; err != nil {
		return s.captureError(errors.Wrap(err, "failed to exec query"))
	}

	return nil
}

func (s *SQL) RawSql(sql string, object interface{}, args ...interface{}) error {
	defer s.doMonitor("RAW SQL", s.connection.DBName, object.(schema.Tabler).TableName())()
	err := s.db.Raw(sql, args).Scan(object).Error
	if err != nil {
		return s.captureError(errors.Wrap(err, "failed to fetch raw query result"))
	}

	return nil
}

func (s *SQL) FindByQuery(tableName string, selectFields []string, query []sql.Query, result interface{}) sql.ORM {
	defer s.doMonitor("FIND BY QUERY", s.connection.DBName, tableName)()
	db := s.db.Table(tableName)
	if len(selectFields) > 0 {
		db = db.Select(selectFields)
	}

	for _, q := range query {
		db = db.Where(q.Field+" "+q.Operator+" (?)", q.Value)
	}

	if s.DryRun {
		res := db.Session(&gorm.Session{DryRun: true}).Find(result)
		s.Logger.Debug(res.Statement.SQL.String())
	}
	res := db.Find(result)

	if err := res.Error; err != nil {
		err = errors.Wrapf(err, "failed to find by query %s", result)
		s.Logger.Error(s.captureError(err))
		return &SQL{
			Logger:         s.Logger,
			db:             db,
			DryRun:         s.DryRun,
			isMonitor:      s.isMonitor,
			monitor:        s.monitor,
			isCaptureError: s.isCaptureError,
			ctx:            s.ctx,
			connection:     s.connection,
			requestId:      s.requestId,
		}
	}

	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) FillQuery(tableName string, query []sql.Query) sql.ORM {
	db := s.db.Table(tableName)

	for _, q := range query {
		db = db.Where(q.Field+" "+q.Operator+" (?)", q.Value)
	}

	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Count() int64 {
	var (
		count = int64(0)
	)

	s.db.Count(&count)
	return count
}

func (s *SQL) Select(fields []string) sql.ORM {
	db := s.db

	for _, field := range fields {
		db = db.Select(field)
	}

	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Sort(sorts sql.Sorts) sql.ORM {
	db := s.db
	for _, sort := range sorts {
		if strings.ToLower(sort.Direction) != sql.DESC {
			sort.Direction = sql.ASC
		}
		db.Order(fmt.Sprintf("%s %s", sort.Field, sort.Direction))
	}
	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Join(joins sql.Joins) sql.ORM {
	db := s.db
	for _, join := range joins {
		if len(join.On) == 0 {
			db = db.Joins(fmt.Sprintf(sql.JoinTemplate, join.Type, join.Table, ""))
			continue
		}

		joinQuery := make([]string, 0)
		for k, q := range join.On {
			prefix := "AND"
			if k == 0 {
				prefix = "ON"
			}
			joinQuery = append(joinQuery, fmt.Sprintf(sql.JoinOnTemplate, prefix, q.Local, q.Operator, fmt.Sprintf("%s.%s", join.Table, q.Foreign)))
		}
		db = db.Joins(fmt.Sprintf(sql.JoinTemplate, join.Type, join.Table, strings.Join(joinQuery, " ")))
	}
	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Group(by []string) sql.ORM {
	db := s.db

	for _, group := range by {
		db = db.Group(group)
	}

	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

/*
========================================
MIGRATOR SECTION
========================================
*/

func (s *SQL) HasTable(name string) bool {
	return s.db.Migrator().HasTable(name)
}

func (s *SQL) CreateTable(data interface{}) error {
	err := s.db.Migrator().CreateTable(data)
	if err != nil {
		return s.captureError(errors.Wrap(err, "failed to create table"))
	}
	return nil
}

func (s *SQL) CreateTableWithName(name string, data interface{}) error {
	err := s.db.Table(name).Migrator().CreateTable(data)
	if err != nil {
		return s.captureError(errors.Wrap(err, fmt.Sprintf("failed to create table %s", name)))
	}
	return nil
}

func (s *SQL) DropTable(data interface{}) error {
	err := s.db.Migrator().DropTable(data)
	if err != nil {
		return s.captureError(errors.Wrap(err, "failed to drop table"))
	}
	return nil
}

func (s *SQL) DropTableWithName(name string, data interface{}) error {
	err := s.db.Table(name).Migrator().DropTable(data)
	if err != nil {
		return s.captureError(errors.Wrap(err, fmt.Sprintf("failed to drop table %s", name)))
	}
	return nil
}

func (s *SQL) HasColumn(data interface{}, column string) bool {
	return s.db.Migrator().HasColumn(data, column)

}

func (s *SQL) AddColumn(data interface{}, column string) error {
	return s.db.Migrator().AddColumn(data, column)
}

func (s *SQL) DropColumn(data interface{}, column string) error {
	return s.db.Migrator().DropColumn(data, column)
}

func (s *SQL) AlterColumn(data interface{}, column string) error {
	return s.db.Migrator().AlterColumn(data, column)
}

func (s *SQL) RowsAffected() int64 {

	return s.db.RowsAffected
}

/*
========================================
TRANSACTION SECTION
========================================
*/

func (s *SQL) Table(name string) sql.ORM {
	db := s.db.Table(name)
	return &SQL{
		Logger:         s.Logger,
		db:             db,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Begin() sql.ORM {
	db := s.db
	dbTrx := db.Begin()
	return &SQL{
		Logger:         s.Logger,
		db:             dbTrx,
		DryRun:         s.DryRun,
		isMonitor:      s.isMonitor,
		monitor:        s.monitor,
		isCaptureError: s.isCaptureError,
		ctx:            s.ctx,
		connection:     s.connection,
		requestId:      s.requestId,
	}
}

func (s *SQL) Commit() error {
	err := s.db.Commit().Error
	if err != nil {
		return s.captureError(errors.Wrap(err, "commit failed"))
	}
	return nil
}

func (s *SQL) Rollback() error {
	err := s.db.Rollback().Error
	if err != nil {
		return s.captureError(errors.Wrap(err, "rollback failed"))
	}
	return nil
}

func (s *SQL) Ping() error {
	db, err := s.db.DB()
	if err != nil {
		return s.captureError(err)
	}
	return db.Ping()
}

func (s *SQL) Close() error {
	db, err := s.db.DB()
	if err != nil {
		return s.captureError(err)
	}
	return db.Close()
}

func (s *SQL) Error() error {
	return s.captureError(s.db.Error)
}

func (s *SQL) Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) sql.ORM {
	return &SQL{
		Logger:         s.Logger,
		db:             s.db,
		DryRun:         s.DryRun,
		isMonitor:      true,
		monitor:        mntr,
		ctx:            ctx,
		requestId:      requestId,
		isCaptureError: captureError,
	}
}

func (s *SQL) doMonitor(action string, database, table string) func() {
	if s.isMonitor {
		tr := s.startMonitor(action, database, table)
		return func() {
			s.finishMonitor(tr)
		}
	}
	return func() {}
}

func (s *SQL) finishMonitor(transaction monitor.Transaction) {
	transaction.Finish()
}

func (s *SQL) startMonitor(action string, database, table string) monitor.Transaction {
	tags := []monitor.Tag{
		{"requestId", s.requestId},
		{"action", action},
		{"database", database},
		{"table", table},
	}

	return s.monitor.NewTransactionFromContext(s.ctx, monitor.Tick{
		Operation:       "db",
		TransactionName: action,
		Tags:            tags,
	})
}

func (s *SQL) captureError(err error) error {
	if s.isCaptureError {
		s.monitor.Capture(err)
	}
	return err
}

func getTableName(data interface{}) string {
	if table, ok := data.(schema.Tabler); ok {
		return table.TableName()
	}
	return "unknown"
}
