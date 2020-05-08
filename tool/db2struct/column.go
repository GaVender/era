package db2struct

import (
	"fmt"
	"regexp"
)

const (
	matchBrackets = `\(.*\)`

	nullYes = "YES"
	nullNo  = "NO"
)

var (
	columnMap = map[string]string{
		"tinyint":   "int8",
		"smallint":  "int16",
		"mediumint": "int32",
		"int":       "int",
		"bigint":    "int64",

		"tinyint unsigned":   "uint8",
		"smallint unsigned":  "uint16",
		"mediumint unsigned": "uint32",
		"int unsigned":       "uint",
		"bigint unsigned":    "uint64",

		"float":            "float32",
		"float unsigned":   "float64",
		"double":           "float64",
		"double unsigned":  "float64",
		"decimal":          "float64",
		"decimal unsigned": "float64",

		"char":       "string",
		"varchar":    "string",
		"tinytext":   "string",
		"text":       "string",
		"mediumtext": "string",
		"longtext":   "string",

		"tinyblob":   "[]byte",
		"blob":       "[]byte",
		"mediumblob": "[]byte",
		"longblob":   "[]byte",

		"date":      "time.Time",
		"datetime":  "time.Time",
		"timestamp": "time.Time",

		"json": "types.JSONText",
	}

	matchBracketsRegex = regexp.MustCompile(matchBrackets)
)

func columnMapper(s string, nullable string) string {
	s = matchBracketsRegex.ReplaceAllString(s, "")
	if t, ok := columnMap[s]; ok {
		return nullableField(t, nullable)
	}

	panic(fmt.Errorf("unknown column type: %s", s))
}

func nullableField(field, nullable string) string {
	if nullable == nullYes {
		switch field {
		case "int8", "int16", "int32":
			return "sql.NullInt32"
		case "int", "int64":
			return "sql.NullInt64"
		case "float32", "float64":
			return "sql.NullFloat64"
		case "string":
			return "sql.NullString"
		case "time.Time":
			return "sql.NullTime"
		}
	}

	return field
}
