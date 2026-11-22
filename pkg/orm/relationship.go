package orm

import (
	"reflect"
	"strings"
	"time"
)

// RelationType defines the structural association between two database models.
type RelationType string

const (
	RelationHasOne     RelationType = "has_one"
	RelationHasMany    RelationType = "has_many"
	RelationBelongsTo  RelationType = "belongs_to"
	RelationManyToMany RelationType = "many_to_many"
)

// Relationship represents association metadata between an owning model and a related model.
type Relationship struct {
	Name           string       // Struct field name, e.g. "Profile", "Posts", "Author"
	Type           RelationType // HasOne, HasMany, BelongsTo, ManyToMany
	Field          reflect.StructField
	FieldType      reflect.Type // Target model struct type (e.g. Post, Profile)
	IsSlice        bool         // True if target field is a slice ([]T or []*T)
	IsPtr          bool         // True if target field or slice element is pointer
	ForeignKey     string       // Foreign key column name
	References     string       // Referenced primary key column name (default "id")
	JoinTable      string       // Pivot table name for ManyToMany associations
	JoinForeignKey string       // Pivot table column referencing owner model
	JoinReferences string       // Pivot table column referencing target model
}

func isRelationshipField(field reflect.StructField, tag string) bool {
	lowerTag := strings.ToLower(tag)
	if strings.Contains(lowerTag, "has_one") ||
		strings.Contains(lowerTag, "has_many") ||
		strings.Contains(lowerTag, "belongs_to") ||
		strings.Contains(lowerTag, "many_to_many") {
		return true
	}

	ft := field.Type
	if ft.Kind() == reflect.Slice {
		elem := ft.Elem()
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		return elem.Kind() == reflect.Struct && elem != reflect.TypeOf(time.Time{})
	}

	if ft.Kind() == reflect.Ptr {
		elem := ft.Elem()
		return elem.Kind() == reflect.Struct && elem != reflect.TypeOf(time.Time{})
	}

	return false
}

func parseRelationship(tm *TableModel, field reflect.StructField, tag string) *Relationship {
	fieldType := field.Type
	isSlice := false
	isPtr := false

	if fieldType.Kind() == reflect.Slice {
		isSlice = true
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() == reflect.Ptr {
		isPtr = true
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() != reflect.Struct {
		return nil
	}
	if fieldType == reflect.TypeOf(time.Time{}) {
		return nil
	}

	parts := strings.Split(tag, ";")
	relType := RelationType("")
	foreignKey := ""
	references := "id"
	joinTable := ""
	joinForeignKey := ""
	joinReferences := "id"

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		k := strings.ToLower(strings.TrimSpace(kv[0]))
		v := ""
		if len(kv) == 2 {
			v = strings.TrimSpace(kv[1])
		}

		switch k {
		case "has_one":
			relType = RelationHasOne
		case "has_many":
			relType = RelationHasMany
		case "belongs_to":
			relType = RelationBelongsTo
		case "many_to_many":
			relType = RelationManyToMany
			if v != "" {
				joinTable = v
			}
		case "join_table":
			joinTable = v
		case "foreign_key", "fk":
			foreignKey = v
		case "references", "ref":
			references = v
		case "join_foreign_key", "join_fk":
			joinForeignKey = v
		case "join_references", "join_ref":
			joinReferences = v
		}
	}

	// Auto-infer relationship type if not explicitly tagged
	if relType == "" {
		if isSlice {
			relType = RelationHasMany
		} else {
			fkGuess := toSnakeCase(field.Name) + "_id"
			if _, exists := tm.ColumnMap[strings.ToLower(fkGuess)]; exists {
				relType = RelationBelongsTo
				foreignKey = fkGuess
			} else {
				relType = RelationHasOne
			}
		}
	}

	// Default foreign keys
	if foreignKey == "" {
		if relType == RelationBelongsTo {
			foreignKey = toSnakeCase(field.Name) + "_id"
		} else {
			ownerSingular := strings.TrimSuffix(tm.TableName, "s")
			foreignKey = ownerSingular + "_id"
		}
	}

	if joinTable == "" && relType == RelationManyToMany {
		ownerSingular := strings.TrimSuffix(tm.TableName, "s")
		targetSingular := strings.TrimSuffix(toSnakeCase(fieldType.Name()), "s")
		joinTable = ownerSingular + "_" + targetSingular + "s"
	}
	if joinForeignKey == "" && relType == RelationManyToMany {
		ownerSingular := strings.TrimSuffix(tm.TableName, "s")
		joinForeignKey = ownerSingular + "_id"
	}
	if joinReferences == "" && relType == RelationManyToMany {
		targetSingular := strings.TrimSuffix(toSnakeCase(fieldType.Name()), "s")
		joinReferences = targetSingular + "_id"
	}

	return &Relationship{
		Name:           field.Name,
		Type:           relType,
		Field:          field,
		FieldType:      fieldType,
		IsSlice:        isSlice,
		IsPtr:          isPtr,
		ForeignKey:     foreignKey,
		References:     references,
		JoinTable:      joinTable,
		JoinForeignKey: joinForeignKey,
		JoinReferences: joinReferences,
	}
}
