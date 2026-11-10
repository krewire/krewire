package schema

import (
	"strings"
)

// ForeignKeyDefinition represents a FOREIGN KEY constraint.
type ForeignKeyDefinition struct {
	Column         string
	TargetTable    string
	TargetColumn   string
	OnDeleteAction string
	OnUpdateAction string
}

// References specifies the foreign table column being referenced.
func (f *ForeignKeyDefinition) References(targetColumn string) *ForeignKeyDefinition {
	f.TargetColumn = targetColumn
	return f
}

// On specifies the target table being referenced.
func (f *ForeignKeyDefinition) On(targetTable string) *ForeignKeyDefinition {
	f.TargetTable = targetTable
	return f
}

// OnDelete specifies the ON DELETE action (CASCADE, SET NULL, RESTRICT).
func (f *ForeignKeyDefinition) OnDelete(action string) *ForeignKeyDefinition {
	f.OnDeleteAction = strings.ToUpper(action)
	return f
}

// OnUpdate specifies the ON UPDATE action (CASCADE, SET NULL, RESTRICT).
func (f *ForeignKeyDefinition) OnUpdate(action string) *ForeignKeyDefinition {
	f.OnUpdateAction = strings.ToUpper(action)
	return f
}

// IndexDefinition represents a table-level index or unique constraint.
type IndexDefinition struct {
	Name     string
	Columns  []string
	IsUnique bool
}

// Blueprint represents a table schema specification (mirroring Laravel Blueprint).
type Blueprint struct {
	TableName     string
	Columns       []*ColumnDefinition
	PrimaryKeys   []string
	ForeignKeys   []*ForeignKeyDefinition
	Indexes       []*IndexDefinition
	DropColumns   []string
	DropIndexes   []string
	DropForeigns  []string
	RenameColumns map[string]string // from -> to
	IsCreate      bool
}

// NewBlueprint creates a new Blueprint for a table.
func NewBlueprint(tableName string, isCreate bool) *Blueprint {
	return &Blueprint{
		TableName:     tableName,
		RenameColumns: make(map[string]string),
		IsCreate:      isCreate,
	}
}

// ID creates an auto-incrementing integer or big integer primary key (default: "id").
func (b *Blueprint) ID(name ...string) *ColumnDefinition {
	colName := "id"
	if len(name) > 0 && name[0] != "" {
		colName = name[0]
	}
	col := &ColumnDefinition{
		Name:         colName,
		Type:         TypeID,
		IsPrimaryKey: true,
		IsAutoInc:    true,
		IsNullable:   false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// BigID creates an auto-incrementing 64-bit big integer primary key (default: "id").
func (b *Blueprint) BigID(name ...string) *ColumnDefinition {
	colName := "id"
	if len(name) > 0 && name[0] != "" {
		colName = name[0]
	}
	col := &ColumnDefinition{
		Name:         colName,
		Type:         TypeBigID,
		IsPrimaryKey: true,
		IsAutoInc:    true,
		IsNullable:   false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// UUID creates a UUID column (default: "id" if empty).
func (b *Blueprint) UUID(name ...string) *ColumnDefinition {
	colName := "uuid"
	if len(name) > 0 && name[0] != "" {
		colName = name[0]
	}
	col := &ColumnDefinition{
		Name:       colName,
		Type:       TypeUUID,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// String creates a VARCHAR column with optional length (default 255).
func (b *Blueprint) String(name string, length ...int) *ColumnDefinition {
	l := 255
	if len(length) > 0 && length[0] > 0 {
		l = length[0]
	}
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeString,
		Length:     l,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Text creates a standard TEXT column.
func (b *Blueprint) Text(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeText,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// MediumText creates a MEDIUMTEXT column.
func (b *Blueprint) MediumText(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeMediumText,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// LongText creates a LONGTEXT column.
func (b *Blueprint) LongText(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeLongText,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Integer creates an INTEGER column.
func (b *Blueprint) Integer(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeInteger,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// TinyInt creates a TINYINT column.
func (b *Blueprint) TinyInt(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeTinyInt,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// SmallInt creates a SMALLINT column.
func (b *Blueprint) SmallInt(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeSmallInt,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// BigInteger creates a BIGINT column.
func (b *Blueprint) BigInteger(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeBigInt,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Float creates a single-precision FLOAT column.
func (b *Blueprint) Float(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeFloat,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Double creates a double-precision DOUBLE column.
func (b *Blueprint) Double(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeDouble,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Decimal creates a DECIMAL column with precision and scale.
func (b *Blueprint) Decimal(name string, precision, scale int) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeDecimal,
		Precision:  precision,
		Scale:      scale,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Boolean creates a BOOLEAN column.
func (b *Blueprint) Boolean(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeBoolean,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Date creates a DATE column.
func (b *Blueprint) Date(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeDate,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// DateTime creates a DATETIME column.
func (b *Blueprint) DateTime(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeDateTime,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Timestamp creates a TIMESTAMP column.
func (b *Blueprint) Timestamp(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeTimestamp,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Timestamps creates created_at and updated_at TIMESTAMP columns (like Laravel).
func (b *Blueprint) Timestamps() {
	b.Timestamp("created_at").Nullable().DefaultRaw("CURRENT_TIMESTAMP")
	b.Timestamp("updated_at").Nullable().DefaultRaw("CURRENT_TIMESTAMP")
}

// SoftDeletes creates a deleted_at TIMESTAMP NULL column for soft deletion.
func (b *Blueprint) SoftDeletes() *ColumnDefinition {
	return b.Timestamp("deleted_at").Nullable()
}

// RememberToken creates a nullable remember_token VARCHAR(100) column (like Laravel).
func (b *Blueprint) RememberToken() *ColumnDefinition {
	return b.String("remember_token", 100).Nullable()
}

// JSON creates a JSON / JSONB column.
func (b *Blueprint) JSON(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeJSON,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Binary creates a BLOB / BYTEA column.
func (b *Blueprint) Binary(name string) *ColumnDefinition {
	col := &ColumnDefinition{
		Name:       name,
		Type:       TypeBinary,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// Foreign creates a foreign key constraint definition.
func (b *Blueprint) Foreign(column string) *ForeignKeyDefinition {
	fk := &ForeignKeyDefinition{Column: column}
	b.ForeignKeys = append(b.ForeignKeys, fk)
	return fk
}

// PrimaryKey sets multi-column primary keys.
func (b *Blueprint) PrimaryKey(columns ...string) {
	b.PrimaryKeys = columns
}

// Index creates a table-level secondary index across columns.
func (b *Blueprint) Index(name string, columns ...string) {
	b.Indexes = append(b.Indexes, &IndexDefinition{
		Name:     name,
		Columns:  columns,
		IsUnique: false,
	})
}

// Unique creates a table-level unique index across columns.
func (b *Blueprint) Unique(name string, columns ...string) {
	b.Indexes = append(b.Indexes, &IndexDefinition{
		Name:     name,
		Columns:  columns,
		IsUnique: true,
	})
}

// DropColumn marks columns to be dropped in an ALTER TABLE.
func (b *Blueprint) DropColumn(names ...string) {
	b.DropColumns = append(b.DropColumns, names...)
}

// DropIndex marks an index to be dropped.
func (b *Blueprint) DropIndex(name string) {
	b.DropIndexes = append(b.DropIndexes, name)
}

// DropForeign marks a foreign key constraint to be dropped.
func (b *Blueprint) DropForeign(name string) {
	b.DropForeigns = append(b.DropForeigns, name)
}

// RenameColumn renames a column from oldName to newName.
func (b *Blueprint) RenameColumn(from, to string) {
	b.RenameColumns[from] = to
}
