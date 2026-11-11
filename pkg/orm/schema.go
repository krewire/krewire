package orm

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/krewire/krewire/pkg/errs"
)

// Column represents a single mapped field of a model struct.
type Column struct {
	Name         string
	FieldName    string
	Type         reflect.Type
	IsPrimaryKey bool
	IsAutoInc    bool
	IsNullable   bool
	DefaultValue string
	Unique       bool
	IsSoftDelete bool
}

// TableModel represents the introspection metadata of a database table model.
type TableModel struct {
	TableName       string
	ModelType       reflect.Type
	Columns         []*Column
	ColumnMap       map[string]*Column // key: column name lowercased
	PrimaryKey      *Column
	CreatedAtCol    *Column
	UpdatedAtCol    *Column
	DeletedAtCol    *Column
	IsSoftDeletable bool
}

var (
	modelCache sync.Map // map[reflect.Type]*TableModel
)

// TableNamer is an optional interface models can implement to customize their table name.
type TableNamer interface {
	TableName() string
}

// ParseModel introspects a Go struct or pointer to struct and returns its TableModel metadata.
func ParseModel(v interface{}) (*TableModel, error) {
	if v == nil {
		return nil, errs.UsageError("orm: model cannot be nil")
	}

	typ := reflect.TypeOf(v)
	for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, errs.UsageError(fmt.Sprintf("orm: model must be a struct or slice of structs, got %s", typ.Kind()))
	}

	if cached, ok := modelCache.Load(typ); ok {
		return cached.(*TableModel), nil
	}

	tableName := toSnakeCase(typ.Name())
	if namer, ok := reflect.New(typ).Interface().(TableNamer); ok {
		tableName = namer.TableName()
	}

	tm := &TableModel{
		TableName: tableName,
		ModelType: typ,
		ColumnMap: make(map[string]*Column),
	}

	var inspectStruct func(t reflect.Type)
	inspectStruct = func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			// Ignore unexported fields
			if field.PkgPath != "" {
				continue
			}

			// If embedded anonymous struct, flatten its fields into the model
			if field.Anonymous && field.Type.Kind() == reflect.Struct {
				inspectStruct(field.Type)
				continue
			}

			tag := field.Tag.Get("orm")
			if tag == "-" {
				continue
			}
			if tag == "" {
				tag = field.Tag.Get("db")
				if tag == "-" {
					continue
				}
			}

			col := parseColumnTag(field, tag)
			tm.Columns = append(tm.Columns, col)
			tm.ColumnMap[strings.ToLower(col.Name)] = col

			if col.IsPrimaryKey && tm.PrimaryKey == nil {
				tm.PrimaryKey = col
			}

			// Detect timestamps and soft deletes
			lowerName := strings.ToLower(col.Name)
			if (lowerName == "created_at" || col.FieldName == "CreatedAt") && tm.CreatedAtCol == nil {
				tm.CreatedAtCol = col
			}
			if (lowerName == "updated_at" || col.FieldName == "UpdatedAt") && tm.UpdatedAtCol == nil {
				tm.UpdatedAtCol = col
			}
			if col.IsSoftDelete || lowerName == "deleted_at" || col.FieldName == "DeletedAt" {
				if tm.DeletedAtCol == nil {
					col.IsSoftDelete = true
					tm.DeletedAtCol = col
					tm.IsSoftDeletable = true
				}
			}
		}
	}
	inspectStruct(typ)

	// If no PK was explicitly declared, look for field named "ID"
	if tm.PrimaryKey == nil {
		if col, ok := tm.ColumnMap["id"]; ok {
			col.IsPrimaryKey = true
			tm.PrimaryKey = col
		}
	}

	modelCache.Store(typ, tm)
	return tm, nil
}

func parseColumnTag(field reflect.StructField, tag string) *Column {
	col := &Column{
		Name:       toSnakeCase(field.Name),
		FieldName:  field.Name,
		Type:       field.Type,
		IsNullable: field.Type.Kind() == reflect.Ptr,
	}

	if strings.ToLower(field.Name) == "id" {
		col.IsPrimaryKey = true
		if isIntegerType(field.Type) {
			col.IsAutoInc = true
		}
	}

	if tag == "" {
		return col
	}

	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, ":", 2)
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := ""
		if len(kv) == 2 {
			val = strings.TrimSpace(kv[1])
		}

		switch key {
		case "column":
			if val != "" {
				col.Name = val
			}
		case "pk", "primary_key":
			col.IsPrimaryKey = true
		case "auto", "auto_increment":
			col.IsAutoInc = true
		case "unique":
			col.Unique = true
		case "not_null":
			col.IsNullable = false
		case "null":
			col.IsNullable = true
		case "default":
			col.DefaultValue = val
		case "soft_delete", "deleted_at":
			col.IsSoftDelete = true
		default:
			// If tag is just a bare name like `db:"user_id"`
			if len(kv) == 1 && !strings.Contains(part, "=") {
				col.Name = part
			}
		}
	}

	return col
}

func isIntegerType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func toSnakeCase(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			if prev >= 'a' && prev <= 'z' {
				b.WriteRune('_')
			} else if i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				b.WriteRune('_')
			}
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
