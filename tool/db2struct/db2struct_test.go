package db2struct

import (
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	type args struct {
		dsn         string
		pkgName     string
		output      string
		tag         string
		skipComment bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			args: args{
				dsn:         "root:root@tcp(127.0.0.1:3306)/HRMS-Role?parseTime=true&loc=Asia%2FShanghai&charset=utf8mb4&collation=utf8mb4_general_ci",
				pkgName:     "db2struct",
				output:      filepath.FromSlash("table.go"),
				tag:         "db",
				skipComment: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			Run(args.dsn, args.pkgName, args.output, args.tag, args.skipComment)
		})
	}
}
