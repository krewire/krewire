package orm

import (
	"database/sql"
	"reflect"
	"strings"
)

// scanRow maps an active database SQL row into the destination struct based on table metadata.
func scanRow(rows *sql.Rows, dest interface{}, tm *TableModel) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	val := reflect.ValueOf(dest)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	scanPointers := make([]interface{}, len(cols))
	for i, colName := range cols {
		col, ok := tm.ColumnMap[strings.ToLower(colName)]
		if ok {
			fieldVal := val.FieldByName(col.FieldName)
			if fieldVal.IsValid() && fieldVal.CanAddr() {
				scanPointers[i] = fieldVal.Addr().Interface()
				continue
			}
		}
		var dummy interface{}
		scanPointers[i] = &dummy
	}

	return rows.Scan(scanPointers...)
}
