package schema

import (
	"fmt"
)

// ColumnType represents the logical data type of a column.
type ColumnType string

const (
	TypeID         ColumnType = "id"
	TypeBigID      ColumnType = "bigid"
	TypeUUID       ColumnType = "uuid"
	TypeString     ColumnType = "string"
	TypeText       ColumnType = "text"
	TypeMediumText ColumnType = "mediumtext"
	TypeLongText   ColumnType = "longtext"
	TypeInteger    ColumnType = "integer"
	TypeTinyInt    ColumnType = "tinyint"
	TypeSmallInt   ColumnType = "smallint"
	TypeBigInt     ColumnType = "bigint"
	TypeFloat      ColumnType = "float"
	TypeDouble     ColumnType = "double"
	TypeDecimal    ColumnType = "decimal"
	TypeBoolean    ColumnType = "boolean"
	TypeDate       ColumnType = "date"
	TypeDateTime   ColumnType = "datetime"
	TypeTimestamp  ColumnType = "timestamp"
	TypeJSON       ColumnType = "json"
	TypeBinary     ColumnType = "binary"
)

// ColumnDefinition represents the fluent definition of a database column.
type ColumnDefinition struct {
	Name         string
	Type         ColumnType
	Length       int
	Precision    int
	Scale        int
	IsPrimaryKey bool
	IsAutoInc    bool
	IsNullable   bool
	IsUnique     bool
	HasIndex     bool
	IndexName    string
	IsUnsigned   bool
	DefaultVal   *string
	CommentText  string
}

// Nullable marks the column as allowing NULL values.
func (c *ColumnDefinition) Nullable() *ColumnDefinition {
	c.IsNullable = true
	return c
}

// NotNull marks the column as NOT NULL (default for non-nullable types).
func (c *ColumnDefinition) NotNull() *ColumnDefinition {
	c.IsNullable = false
	return c
}

// Unique marks the column as having a UNIQUE constraint.
func (c *ColumnDefinition) Unique(name ...string) *ColumnDefinition {
	c.IsUnique = true
	if len(name) > 0 {
		c.IndexName = name[0]
	}
	return c
}

// Index marks the column as having a secondary index.
func (c *ColumnDefinition) Index(name ...string) *ColumnDefinition {
	c.HasIndex = true
	if len(name) > 0 {
		c.IndexName = name[0]
	}
	return c
}

// Primary marks the column as part of the PRIMARY KEY.
func (c *ColumnDefinition) Primary() *ColumnDefinition {
	c.IsPrimaryKey = true
	c.IsNullable = false
	return c
}

// Unsigned marks an integer/numeric column as UNSIGNED.
func (c *ColumnDefinition) Unsigned() *ColumnDefinition {
	c.IsUnsigned = true
	return c
}

// Default sets a default value for the column.
func (c *ColumnDefinition) Default(val interface{}) *ColumnDefinition {
	var s string
	switch v := val.(type) {
	case string:
		s = fmt.Sprintf("'%s'", v)
	case bool:
		if v {
			s = "1"
		} else {
			s = "0"
		}
	default:
		s = fmt.Sprintf("%v", v)
	}
	c.DefaultVal = &s
	return c
}

// DefaultRaw sets a raw SQL default expression (e.g. CURRENT_TIMESTAMP).
func (c *ColumnDefinition) DefaultRaw(rawExpr string) *ColumnDefinition {
	c.DefaultVal = &rawExpr
	return c
}

// Comment sets a database comment on the column.
func (c *ColumnDefinition) Comment(comment string) *ColumnDefinition {
	c.CommentText = comment
	return c
}
