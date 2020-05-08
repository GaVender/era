package db2struct

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/Masterminds/sprig"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"golang.org/x/tools/imports"

	"github.com/GaVender/era/utils"
)

type (
	TemplateData struct {
		PkgName string
		Tables  []Table
	}

	Table struct {
		Name    string `db:"TABLE_NAME"`
		Comment string `db:"TABLE_COMMENT"`
		Columns []Column
		Indexes map[string][]Index
	}

	Column struct {
		Field   string         `db:"Field"`
		Type    string         `db:"Type"`
		Null    string         `db:"Null"`
		Key     string         `db:"Key"`
		Default sql.NullString `db:"Default"`
		Extra   string         `db:"Extra"`
		Comment string         `db:"Comment"`
		Tag     template.HTML
	}

	Index struct {
		KeyName    string `db:"Key_name"`
		ColumnName string `db:"Column_name"`
		NonUnique  uint8  `db:"Non_unique"`
	}
)

const command = "db2struct"

var (
	_tmplFile    string
	_dsn         string
	_pkgName     string
	_output      string
	_tag         string
	_skipComment bool
)

func init() {
	dir, _ := os.Getwd()
	_tmplFile = string(utils.MustReadFile(dir + `\` + command + ".tmpl"))
}

func Run(dsn, pkgName, output, tag string, skipComment bool) {
	_dsn = dsn
	_pkgName = pkgName
	_output = output
	_tag = tag
	_skipComment = skipComment

	var (
		db        = sqlx.MustOpen("mysql", _dsn)
		db2struct = TemplateData{PkgName: _pkgName}
	)

	db2struct.Tables = getTables(db)
	if err := db.Close(); err != nil {
		panic(fmt.Errorf("db close: %w", err))
	}

	t, err := template.New(command).Funcs(sprig.FuncMap()).Funcs(map[string]interface{}{
		"snake2camel":  snake2camel,
		"columnMapper": columnMapper,
	}).Parse(_tmplFile)
	if err != nil {
		panic(fmt.Errorf("template parse: %w", err))
	}

	var buf bytes.Buffer
	t = t.Lookup(command)
	if err := t.Execute(&buf, db2struct); err != nil {
		panic(fmt.Errorf("template execute: %w", err))
	}

	b, err := imports.Process(_output, buf.Bytes(), nil)
	if err != nil {
		panic(fmt.Errorf("goimports: %w", err))
	}

	if err := ioutil.WriteFile(_output, b, 0666); err != nil {
		panic(fmt.Errorf("write template to filer: %w", err))
	}
}

func getCurrentDB(db *sqlx.DB) string {
	var dbName string

	if err := db.Get(&dbName, `SELECT DATABASE();`); err != nil {
		panic(fmt.Errorf("select current database: %w", err))
	}

	if len(dbName) == 0 {
		panic("database is not selected")
	}

	return dbName
}

func getTables(db *sqlx.DB) []Table {
	var (
		tables []Table
		currDB = getCurrentDB(db)
	)

	rows, err := db.Queryx(`SELECT TABLE_NAME, TABLE_COMMENT FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ?`, currDB)
	if err != nil {
		panic(fmt.Errorf("exec show tables: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		var table Table
		if err := rows.StructScan(&table); err != nil {
			panic(fmt.Errorf("scan table name: %w", err))
		}

		if len(table.Comment) == 0 && !_skipComment {
			panic(fmt.Sprintf("table %s is missing comment", table.Name))
		}

		getIndex(db, &table)
		getColumns(db, &table)
		tables = append(tables, table)
	}

	return tables
}

func getIndex(db *sqlx.DB, table *Table) {
	udb := db.Unsafe()
	rows, err := udb.Queryx("SHOW INDEX FROM " + table.Name)
	if err != nil {
		panic(fmt.Errorf("exec show index %s : %w", table.Name, err))
	}
	defer rows.Close()

	indexMap := make(map[string][]Index)
	for rows.Next() {
		var i Index
		if err := rows.StructScan(&i); err != nil {
			panic(fmt.Errorf("scan table %s index : %w", table.Name, err))
		}

		indexMap[i.ColumnName] = append(indexMap[i.ColumnName], i)
	}

	table.Indexes = indexMap
}

func getColumns(db *sqlx.DB, table *Table) {
	udb := db.Unsafe()
	rows, err := udb.Queryx("SHOW FULL COLUMNS FROM " + table.Name)
	if err != nil {
		panic(fmt.Errorf("exec show columns %s : %w", table.Name, err))
	}
	defer rows.Close()

	for rows.Next() {
		var c Column
		if err := rows.StructScan(&c); err != nil {
			panic(fmt.Errorf("scan table %s columns : %w", table.Name, err))
		}

		if len(c.Comment) == 0 && !_skipComment {
			panic(fmt.Sprintf("table %s.%s is missing comment, please make sure all fields have comment", table.Name, c.Field))
		}

		c.Tag = generateTag(_tag, c, table.Indexes)
		table.Columns = append(table.Columns, c)
	}
}

func generateTag(tag string, column Column, indexes map[string][]Index) template.HTML {
	var buf bytes.Buffer
	buf.WriteString("`")
	buf.WriteString(tag + `:`)
	buf.WriteString(`"`)

	refType := reflect.TypeOf(column)
	refVal := reflect.ValueOf(column)
	for i := 0; i < refVal.NumField(); i++ {
		val := refVal.Field(i).String()

		switch refType.Field(i).Name {
		case "Field":
			buf.WriteString(`Column:` + val + `;`)
		case "Type":
			buf.WriteString(`Type:` + val + `;`)
		case "Null":
			if val == nullNo {
				buf.WriteString(`NOT NULL;`)
			}
		case "Key":
			if val == "PRI" {
				buf.WriteString(`PRIMARY_KEY;`)
			}
			if colIndexes, ok := indexes[column.Field]; ok {
				for _, index := range colIndexes {
					if index.NonUnique == 0 && index.KeyName != "PRIMARY" {
						buf.WriteString(`UNIQUE_INDEX:` + index.KeyName + `;`)
					} else if index.NonUnique == 1 {
						buf.WriteString(`INDEX:` + index.KeyName + `;`)
					}
				}
			}
		case "Default":
			if (refVal.Field(i).Interface()).(sql.NullString).Valid {
				buf.WriteString(`DEFAULT:` + (refVal.Field(i).Interface()).(sql.NullString).String + ";")
			}
		case "Extra":
			if val == "auto_increment" {
				buf.WriteString("AUTO_INCREMENT;")
			}
		}
	}

	if tag == "json" && column.Null == "YES" {
		buf.WriteString(`,omitempty`)
	}

	buf.WriteString(`"`)
	buf.WriteString("`")
	return template.HTML(buf.String())
}
