package orm

import (
	"context"
	"reflect"
	"time"
)

// applyInsertTimestamps automatically sets CreatedAt and UpdatedAt fields if zero.
func applyInsertTimestamps(val reflect.Value, tm *TableModel, now time.Time) {
	if tm.CreatedAtCol != nil {
		field := val.FieldByName(tm.CreatedAtCol.FieldName)
		if field.IsValid() && field.CanSet() {
			if t, ok := field.Interface().(time.Time); ok && t.IsZero() {
				field.Set(reflect.ValueOf(now))
			}
		}
	}
	if tm.UpdatedAtCol != nil {
		field := val.FieldByName(tm.UpdatedAtCol.FieldName)
		if field.IsValid() && field.CanSet() {
			if t, ok := field.Interface().(time.Time); ok && t.IsZero() {
				field.Set(reflect.ValueOf(now))
			}
		}
	}
}

// applyUpdateTimestamps automatically sets UpdatedAt to the current timestamp.
func applyUpdateTimestamps(val reflect.Value, tm *TableModel, now time.Time) {
	if tm.UpdatedAtCol != nil {
		field := val.FieldByName(tm.UpdatedAtCol.FieldName)
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(now))
		}
	}
}

// runBeforeCreateHooks triggers BeforeSave and BeforeCreate lifecycle hooks.
func runBeforeCreateHooks(ctx context.Context, record interface{}) error {
	if hook, ok := record.(BeforeSaver); ok {
		if err := hook.BeforeSave(ctx); err != nil {
			return err
		}
	}
	if hook, ok := record.(BeforeCreator); ok {
		if err := hook.BeforeCreate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// runAfterCreateHooks triggers AfterCreate and AfterSave lifecycle hooks.
func runAfterCreateHooks(ctx context.Context, record interface{}) error {
	if hook, ok := record.(AfterCreator); ok {
		if err := hook.AfterCreate(ctx); err != nil {
			return err
		}
	}
	if hook, ok := record.(AfterSaver); ok {
		if err := hook.AfterSave(ctx); err != nil {
			return err
		}
	}
	return nil
}

// runBeforeUpdateHooks triggers BeforeSave and BeforeUpdate lifecycle hooks.
func runBeforeUpdateHooks(ctx context.Context, record interface{}) error {
	if hook, ok := record.(BeforeSaver); ok {
		if err := hook.BeforeSave(ctx); err != nil {
			return err
		}
	}
	if hook, ok := record.(BeforeUpdater); ok {
		if err := hook.BeforeUpdate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// runAfterUpdateHooks triggers AfterUpdate and AfterSave lifecycle hooks.
func runAfterUpdateHooks(ctx context.Context, record interface{}) error {
	if hook, ok := record.(AfterUpdater); ok {
		if err := hook.AfterUpdate(ctx); err != nil {
			return err
		}
	}
	if hook, ok := record.(AfterSaver); ok {
		if err := hook.AfterSave(ctx); err != nil {
			return err
		}
	}
	return nil
}

// runBeforeDeleteHooks triggers BeforeDelete lifecycle hook.
func runBeforeDeleteHooks(ctx context.Context, record interface{}) error {
	if hook, ok := record.(BeforeDeleter); ok {
		return hook.BeforeDelete(ctx)
	}
	return nil
}

// runAfterDeleteHooks triggers AfterDelete lifecycle hook.
func runAfterDeleteHooks(ctx context.Context, record interface{}) error {
	if hook, ok := record.(AfterDeleter); ok {
		return hook.AfterDelete(ctx)
	}
	return nil
}
