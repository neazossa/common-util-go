package postgres

import (
	"fmt"
)

func getImportTemplate(packageName string) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/neazzosa/common-util-go/persistent/sql/sql"
	"github.com/neazzosa/common-util-go/tools/migrations/postgres"
)
`, packageName)
}

func getScriptTemplate(timestamp string) string {
	return fmt.Sprintf(`func init() {
	script["%s"] = postgres.SQLMigration{
		Up   : func(orm sql.ORM) error {
			return orm.CreateTable(StructName{})
		},
		Down: func(orm sql.ORM) error {
			return orm.DropTable(StructName{})
		},
	}
}`, timestamp)
}
