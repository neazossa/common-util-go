package postgres

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/neazzosa/common-util-go/persistent/sql/sql"
	"github.com/pkg/errors"
)

const (
	MigrationTableName        = "migrations"
	MigrationTimestamp        = "20060102150405"
	MigrationFileNameTemplate = "%s_%s.go" //[timestamp]_[name]
)

type (
	Migrator interface {
		Up() error
		Down(downToVersion string) error
		Init() error
		Create(name string) error
	}

	migrator struct {
		Orm         sql.ORM
		PackageName string
		CustomPath  string
		Scripts     map[string]SQLMigration
	}

	Migration struct {
		Version   string    `gorm:"version"`
		CreatedAt time.Time `gorm:"created_at"`
	}

	Migrations []Migration

	SQLScript func(sql.ORM) error

	SQLMigration struct {
		Up   SQLScript
		Down SQLScript
	}
)

func NewMigrator(orm sql.ORM, packageName string, customPath string, scripts map[string]SQLMigration) Migrator {
	return &migrator{
		Orm:         orm,
		PackageName: packageName,
		CustomPath:  customPath,
		Scripts:     scripts,
	}
}

func (m *Migration) TableName() string {
	return MigrationTableName
}

func (m *Migrations) TableName() string {
	return MigrationTableName
}

func (m *migrator) Init() error {

	if m.Orm.HasTable(MigrationTableName) {
		return errors.New("migrations table already exist")
	}

	err := m.Orm.CreateTable(&Migration{})
	if err != nil {
		return err
	}
	return nil
}

func (m *migrator) Up() error {
	if !m.Orm.HasTable(MigrationTableName) {
		return errors.New("migrations table not exist")
	}

	if len(m.Scripts) == 0 {
		fmt.Println("no migration to execute")
		return nil
	}

	versions, err := m.getMigratedVersions(sql.ASC)
	if err != nil {
		return err
	}

	for _, version := range versions {
		if _, ok := m.Scripts[version.Version]; ok {
			delete(m.Scripts, version.Version)
		}
	}

	if len(m.Scripts) == 0 {
		fmt.Println("migration already up to date")
		return nil
	}

	keys := make([]string, 0, len(m.Scripts))
	for k := range m.Scripts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, version := range keys {
		fmt.Println("migrating : " + version)
		db := m.Orm.Begin()
		err := m.Scripts[version].Up(db)
		if err != nil {
			_ = db.Rollback()
			fmt.Println("failed migrate : " + version)
			return err
		}

		err = db.Create(&Migration{
			Version:   version,
			CreatedAt: time.Now(),
		})
		if err != nil {
			_ = db.Rollback()
			return err
		}
		_ = db.Commit()
		fmt.Println("migrated : " + version + "\n")
	}

	return nil
}

func (m *migrator) Down(downToVersion string) error {
	if !m.Orm.HasTable(MigrationTableName) {
		return errors.New("migrations table not exist")
	}

	if len(m.Scripts) == 0 {
		fmt.Println("no migration to execute")
		return nil
	}

	if _, ok := m.Scripts[downToVersion]; !ok && downToVersion != "" && downToVersion != "all" {
		return errors.New("unknown migration version")
	}

	versions, err := m.getMigratedVersions(sql.DESC)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		fmt.Println("no migration executed")
		return nil
	}

	for _, version := range versions {
		if downToVersion == version.Version {
			return nil
		}

		fmt.Println("rolling back : " + version.Version)
		db := m.Orm.Begin()
		err := m.Scripts[version.Version].Down(db)
		if err != nil {
			_ = db.Rollback()
			fmt.Println("failed rollback : " + version.Version)
			return err
		}

		err = db.Where("version = ?", version.Version).Delete(version)
		if err != nil {
			_ = db.Rollback()
			return err
		}
		_ = db.Commit()
		fmt.Println("rolled back : " + version.Version + "\n")
	}

	return nil
}

func (m *migrator) Create(name string) error {
	var (
		timestamp = time.Now().Format(MigrationTimestamp)
		fileName  = fmt.Sprintf(MigrationFileNameTemplate, timestamp, name)
	)

	if !m.Orm.HasTable(MigrationTableName) {
		return errors.New("migrations table not exist")
	}

	destination, err := os.Create(m.CustomPath + fileName)
	if err != nil {
		return err
	}

	defer func(destination *os.File) {
		err := destination.Close()
		if err != nil {
			fmt.Println("os.Create:", err)
		}
	}(destination)

	_, err = fmt.Fprintln(destination, getImportTemplate(m.PackageName))
	_, err = fmt.Fprintln(destination, getScriptTemplate(timestamp))

	if err == nil {
		fmt.Println("created migration : ", fileName)
	}
	return err
}

func (m *migrator) getMigratedVersions(direction string) (Migrations, error) {
	var (
		response = Migrations{}
		sorts    = sql.Sorts{
			{
				Field:     "version",
				Direction: direction,
			},
		}
	)

	db := m.Orm.Offset(0).Sort(sorts).FindByQuery(MigrationTableName, []string{}, []sql.Query{}, &response)

	if err := db.Error(); err != nil {
		return response, err
	}

	return response, nil
}
