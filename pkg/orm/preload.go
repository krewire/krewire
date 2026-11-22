package orm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
)

type preloadItem struct {
	relation string
	scope    func(*Query) *Query
}

// Preload registers an association to be eagerly loaded with the query results.
func (q *Query) Preload(relation string, conditions ...func(*Query) *Query) *Query {
	var scope func(*Query) *Query
	if len(conditions) > 0 {
		scope = conditions[0]
	}
	q.preloads = append(q.preloads, preloadItem{
		relation: relation,
		scope:    scope,
	})
	return q
}

func (q *Query) execPreloads(dest interface{}, tm *TableModel) error {
	if len(q.preloads) == 0 || tm == nil || q.db == nil {
		return nil
	}

	parentRecords := extractRecordValues(dest)
	if len(parentRecords) == 0 {
		return nil
	}

	for _, item := range q.preloads {
		rel, ok := tm.RelationshipMap[strings.ToLower(item.relation)]
		if !ok {
			return errs.UsageError(fmt.Sprintf("orm: relationship %q not defined on model %s", item.relation, tm.ModelType.Name()))
		}

		switch rel.Type {
		case RelationHasOne, RelationHasMany:
			if err := q.preloadHas(parentRecords, tm, rel, item.scope); err != nil {
				return err
			}
		case RelationBelongsTo:
			if err := q.preloadBelongsTo(parentRecords, tm, rel, item.scope); err != nil {
				return err
			}
		case RelationManyToMany:
			if err := q.preloadManyToMany(parentRecords, tm, rel, item.scope); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractRecordValues(dest interface{}) []reflect.Value {
	val := reflect.ValueOf(dest)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Slice {
		var list []reflect.Value
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			for elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if elem.IsValid() && elem.Kind() == reflect.Struct {
				list = append(list, elem)
			}
		}
		return list
	}
	if val.Kind() == reflect.Struct {
		return []reflect.Value{val}
	}
	return nil
}

func (q *Query) preloadHas(parents []reflect.Value, parentTM *TableModel, rel *Relationship, scope func(*Query) *Query) error {
	refCol := parentTM.PrimaryKey
	if rel.References != "" && strings.ToLower(rel.References) != "id" {
		if c, ok := parentTM.ColumnMap[strings.ToLower(rel.References)]; ok {
			refCol = c
		}
	}
	if refCol == nil {
		return errs.UsageError("orm: cannot preload relation without referenced column on parent model")
	}

	var parentIDs []interface{}
	seen := make(map[string]bool)
	for _, p := range parents {
		fVal := p.FieldByName(refCol.FieldName)
		if !fVal.IsValid() {
			continue
		}
		id := fVal.Interface()
		key := fmt.Sprintf("%v", id)
		if !seen[key] && !isZeroVal(id) {
			seen[key] = true
			parentIDs = append(parentIDs, id)
		}
	}
	if len(parentIDs) == 0 {
		return nil
	}

	targetPtr := reflect.New(rel.FieldType).Interface()
	targetQuery := q.db.Model(targetPtr)
	if q.tx != nil {
		targetQuery = targetQuery.WithTx(q.tx)
	}
	targetQuery = targetQuery.WhereIn(rel.ForeignKey, parentIDs...)
	if scope != nil {
		targetQuery = scope(targetQuery)
	}

	targetSlicePtr := reflect.New(reflect.SliceOf(reflect.PointerTo(rel.FieldType)))
	if err := targetQuery.FindAll(targetSlicePtr.Interface()); err != nil {
		return err
	}

	targetTM, err := ParseModel(targetPtr)
	if err != nil {
		return err
	}
	targetFKCol := targetTM.ColumnMap[strings.ToLower(rel.ForeignKey)]
	if targetFKCol == nil {
		return errs.UsageError(fmt.Sprintf("orm: foreign key column %q not found on target model %s", rel.ForeignKey, rel.FieldType.Name()))
	}

	targetSlice := targetSlicePtr.Elem()
	grouped := make(map[string][]reflect.Value)
	for i := 0; i < targetSlice.Len(); i++ {
		targetElem := targetSlice.Index(i)
		elemDeref := targetElem
		for elemDeref.Kind() == reflect.Ptr {
			elemDeref = elemDeref.Elem()
		}
		fkVal := elemDeref.FieldByName(targetFKCol.FieldName).Interface()
		key := fmt.Sprintf("%v", fkVal)
		grouped[key] = append(grouped[key], elemDeref)
	}

	for _, p := range parents {
		pkVal := p.FieldByName(refCol.FieldName).Interface()
		key := fmt.Sprintf("%v", pkVal)
		matches := grouped[key]

		fieldVal := p.FieldByName(rel.Name)
		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			continue
		}

		if rel.Type == RelationHasOne {
			if len(matches) > 0 {
				if rel.IsPtr {
					fieldVal.Set(matches[0].Addr())
				} else {
					fieldVal.Set(matches[0])
				}
			}
		} else { // RelationHasMany
			newSlice := reflect.MakeSlice(rel.Field.Type, len(matches), len(matches))
			for j, m := range matches {
				if rel.IsPtr {
					newSlice.Index(j).Set(m.Addr())
				} else {
					newSlice.Index(j).Set(m)
				}
			}
			fieldVal.Set(newSlice)
		}
	}
	return nil
}

func (q *Query) preloadBelongsTo(parents []reflect.Value, parentTM *TableModel, rel *Relationship, scope func(*Query) *Query) error {
	fkCol := parentTM.ColumnMap[strings.ToLower(rel.ForeignKey)]
	if fkCol == nil {
		return errs.UsageError(fmt.Sprintf("orm: foreign key column %q not found on parent model %s", rel.ForeignKey, parentTM.ModelType.Name()))
	}

	var targetIDs []interface{}
	seen := make(map[string]bool)
	for _, p := range parents {
		fVal := p.FieldByName(fkCol.FieldName)
		if !fVal.IsValid() {
			continue
		}
		id := fVal.Interface()
		key := fmt.Sprintf("%v", id)
		if !seen[key] && !isZeroVal(id) {
			seen[key] = true
			targetIDs = append(targetIDs, id)
		}
	}
	if len(targetIDs) == 0 {
		return nil
	}

	targetPtr := reflect.New(rel.FieldType).Interface()
	targetTM, err := ParseModel(targetPtr)
	if err != nil {
		return err
	}
	refCol := targetTM.PrimaryKey
	if rel.References != "" && strings.ToLower(rel.References) != "id" {
		if c, ok := targetTM.ColumnMap[strings.ToLower(rel.References)]; ok {
			refCol = c
		}
	}
	if refCol == nil {
		return errs.UsageError("orm: target model lacks primary key for belongs_to association")
	}

	targetQuery := q.db.Model(targetPtr)
	if q.tx != nil {
		targetQuery = targetQuery.WithTx(q.tx)
	}
	targetQuery = targetQuery.WhereIn(refCol.Name, targetIDs...)
	if scope != nil {
		targetQuery = scope(targetQuery)
	}

	targetSlicePtr := reflect.New(reflect.SliceOf(reflect.PointerTo(rel.FieldType)))
	if err := targetQuery.FindAll(targetSlicePtr.Interface()); err != nil {
		return err
	}

	targetSlice := targetSlicePtr.Elem()
	lookup := make(map[string]reflect.Value)
	for i := 0; i < targetSlice.Len(); i++ {
		targetElem := targetSlice.Index(i)
		elemDeref := targetElem
		for elemDeref.Kind() == reflect.Ptr {
			elemDeref = elemDeref.Elem()
		}
		idVal := elemDeref.FieldByName(refCol.FieldName).Interface()
		lookup[fmt.Sprintf("%v", idVal)] = elemDeref
	}

	for _, p := range parents {
		fkVal := p.FieldByName(fkCol.FieldName).Interface()
		key := fmt.Sprintf("%v", fkVal)
		if match, ok := lookup[key]; ok {
			fieldVal := p.FieldByName(rel.Name)
			if fieldVal.IsValid() && fieldVal.CanSet() {
				if rel.IsPtr {
					fieldVal.Set(match.Addr())
				} else {
					fieldVal.Set(match)
				}
			}
		}
	}
	return nil
}

func (q *Query) preloadManyToMany(parents []reflect.Value, parentTM *TableModel, rel *Relationship, scope func(*Query) *Query) error {
	refCol := parentTM.PrimaryKey
	if refCol == nil {
		return errs.UsageError("orm: many_to_many requires a primary key on owner model")
	}

	var parentIDs []interface{}
	seen := make(map[string]bool)
	for _, p := range parents {
		fVal := p.FieldByName(refCol.FieldName)
		if !fVal.IsValid() {
			continue
		}
		id := fVal.Interface()
		key := fmt.Sprintf("%v", id)
		if !seen[key] && !isZeroVal(id) {
			seen[key] = true
			parentIDs = append(parentIDs, id)
		}
	}
	if len(parentIDs) == 0 {
		return nil
	}

	targetPtr := reflect.New(rel.FieldType).Interface()
	targetTM, err := ParseModel(targetPtr)
	if err != nil {
		return err
	}

	targetTable := targetTM.TableName
	targetRef := targetTM.PrimaryKey.Name

	relQuery := q.db.Table(targetTable)
	if q.tx != nil {
		relQuery = relQuery.WithTx(q.tx)
	}

	joinOn := fmt.Sprintf("%s.%s = %s.%s", rel.JoinTable, rel.JoinReferences, targetTable, targetRef)
	relQuery = relQuery.InnerJoin(rel.JoinTable, joinOn)
	relQuery = relQuery.WhereIn(fmt.Sprintf("%s.%s", rel.JoinTable, rel.JoinForeignKey), parentIDs...)
	if scope != nil {
		relQuery = scope(relQuery)
	}

	relQuery.Select(
		fmt.Sprintf("%s.*", targetTable),
		fmt.Sprintf("%s.%s AS __krew_join_fk", rel.JoinTable, rel.JoinForeignKey),
	)

	querySQL, args := relQuery.builder.BuildSelect()
	if err := relQuery.builder.Error(); err != nil {
		return err
	}

	rows, err := relQuery.query(querySQL, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	grouped := make(map[string][]reflect.Value)
	for rows.Next() {
		newElem := reflect.New(rel.FieldType).Elem()
		joinFK, err := scanManyToManyRow(rows, newElem, targetTM)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("%v", joinFK)
		grouped[key] = append(grouped[key], newElem)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, p := range parents {
		pkVal := p.FieldByName(refCol.FieldName).Interface()
		key := fmt.Sprintf("%v", pkVal)
		matches := grouped[key]

		fieldVal := p.FieldByName(rel.Name)
		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			continue
		}

		newSlice := reflect.MakeSlice(rel.Field.Type, len(matches), len(matches))
		for j, m := range matches {
			if rel.IsPtr {
				newSlice.Index(j).Set(m.Addr())
			} else {
				newSlice.Index(j).Set(m)
			}
		}
		fieldVal.Set(newSlice)
	}
	return nil
}

func scanManyToManyRow(rows *sql.Rows, dest reflect.Value, tm *TableModel) (interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var joinFK interface{}
	scanPointers := make([]interface{}, len(cols))
	for i, colName := range cols {
		if strings.EqualFold(colName, "__krew_join_fk") {
			scanPointers[i] = &joinFK
			continue
		}
		col, ok := tm.ColumnMap[strings.ToLower(colName)]
		if ok {
			fieldVal := dest.FieldByName(col.FieldName)
			if fieldVal.IsValid() && fieldVal.CanAddr() {
				scanPointers[i] = fieldVal.Addr().Interface()
				continue
			}
		}
		var dummy interface{}
		scanPointers[i] = &dummy
	}
	if err := rows.Scan(scanPointers...); err != nil {
		return nil, err
	}
	return joinFK, nil
}

func isZeroVal(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.String:
		return rv.String() == ""
	default:
		return rv.IsZero()
	}
}
