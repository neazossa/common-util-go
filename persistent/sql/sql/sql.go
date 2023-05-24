package sql

import (
	"context"

	"github.com/neazzosa/common-util-go/monitor/monitor"
)

type (
	JoinType string

	Query struct {
		Field    string
		Operator string
		Value    interface{}
	}

	Sort struct {
		Field     string
		Direction string
	}

	On struct {
		Local    string
		Operator string
		Foreign  string
	}

	OnMany []On

	Join struct {
		Type  JoinType
		Table string
		On    OnMany
	}

	Sorts []Sort
	Joins []Join

	OnConflict struct {
		UniqueColumn []string
		OnlyUpdate   []string
	}

	ORM interface {
		Where(interface{}, ...interface{}) ORM
		OrWhere(query interface{}, args ...interface{}) ORM
		First(interface{}) error
		All(interface{}) (int64, error)
		Order(interface{}) ORM
		Limit(int) ORM
		Offset(int) ORM
		Create(interface{}) error
		Update(interface{}) error
		Patch(interface{}, ...string) error
		Delete(interface{}, ...interface{}) error
		Upsert(chunkSize int, data interface{}, onConflict OnConflict) error
		Exec(string, ...interface{}) error
		RawSql(string, interface{}, ...interface{}) error
		FindByQuery(string, []string, []Query, interface{}) ORM
		FillQuery(tableName string, query []Query) ORM
		Count() int64
		Select([]string) ORM
		Sort(sorts Sorts) ORM
		Join(joins Joins) ORM
		Group([]string) ORM
		HasTable(string) bool
		CreateTable(interface{}) error
		CreateTableWithName(string, interface{}) error
		DropTable(interface{}) error
		DropTableWithName(string, interface{}) error
		HasColumn(interface{}, string) bool
		AddColumn(interface{}, string) error
		DropColumn(interface{}, string) error
		AlterColumn(interface{}, string) error
		Table(string) ORM
		Begin() ORM
		Commit() error
		Rollback() error
		Ping() error
		Close() error
		Error() error
		RowsAffected() int64

		Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) ORM
	}

	Connection struct {
		Host     string
		Username string
		Password string
		DBName   string
		Port     string
		SSLMode  string
		Timezone string
		SQLDebug bool
	}
)

const (
	ASC  string = "asc"
	DESC string = "desc"

	JOIN       JoinType = "JOIN"
	LEFT_JOIN  JoinType = "LEFT JOIN"
	RIGHT_JOIN JoinType = "RIGHT JOIN"
	CROSS_JOIN JoinType = "CROSS JOIN"

	JoinTemplate   = "%s %s %s"    // [join type] [foreign table name] [on operation]
	JoinOnTemplate = "%s %s %s %s" // [ON/AND] [local] [operator] [foreign]
)
