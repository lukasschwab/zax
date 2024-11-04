// Package zax provides contextual field logging around the uber-zap logger.

package zax

import (
	"context"

	"go.uber.org/zap"
)

type key string

const (
	loggerKey key = "zax"

	// AbsentFieldsKey is the zap field key used for an array of explicitly-
	// expected keys that couldn't be found in the provided context; see
	// [GetFields].
	AbsentFieldsKey string = "_absentFields"
)

// Set Add passed fields in context
func Set(ctx context.Context, fields []zap.Field) context.Context {
	return context.WithValue(ctx, loggerKey, fields)
}

// Append  appending passed fields to the existing fields in context.
// it's recommended to use Append when you want to append some fields and do not lose the already added fields to context.
func Append(ctx context.Context, fields []zap.Field) context.Context {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		fields = append(loggerFields, fields...)
	}
	return context.WithValue(ctx, loggerKey, fields)
}

// GetAll zap stored fields from context
func GetAll(ctx context.Context) []zap.Field {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		return loggerFields
	}
	return nil
}

// GetFields specified by keys.
func GetFields(ctx context.Context, keys ...string) []zap.Field {
	absentKeys := make([]string, 0, len(keys))
	fields := make([]zap.Field, 0, len(keys))
	for _, key := range keys {
		if field, ok := GetField(ctx, key); ok {
			fields = append(fields, field)
		} else {
			absentKeys = append(absentKeys, key)
		}
	}

	return append(fields, zap.Strings(AbsentFieldsKey, absentKeys))
}

// GetField Get a specific zap stored field from context by key
func GetField(ctx context.Context, key string) (field zap.Field, ok bool) {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		for _, field := range loggerFields {
			if field.Key == key {
				return field, true
			}
		}
	}
	return zap.Field{}, false
}

// Prune overwritten values from the logger context.
func Prune(ctx context.Context) context.Context {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		prunedFields := make([]zap.Field, 0, len(loggerFields))
		seenKeys := map[string]bool{}
		for i := len(loggerFields) - 1; i >= 0; i-- {
			field := loggerFields[i]
			if _, seen := seenKeys[field.Key]; !seen {
				prunedFields = append(prunedFields, field)
			}
			seenKeys[field.Key] = true
		}
		return Set(ctx, prunedFields)
	}
	return ctx
}
