package zax

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type Logger struct {
	logger   *zap.Logger
	recorded *observer.ObservedLogs
	t        *testing.T
}

func NewLogger(t *testing.T) *Logger {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := &Logger{
		logger:   zap.New(core),
		recorded: recorded,
		t:        t,
	}
	return logger
}

func (l *Logger) GetZapLogger() *zap.Logger {
	return l.logger
}

func (l *Logger) GetRecordedLogs() []observer.LoggedEntry {
	return l.recorded.All()
}

func (l *Logger) AssertLogEntryExist(t assert.TestingT, key, value string) bool {
	for _, log := range l.recorded.All() {
		for _, r := range log.Context {
			if r.Key == key && r.String == value {
				return true
			}
		}
	}
	if key == "" && value == "" {
		return true
	}
	return assert.Fail(t, fmt.Sprintf("log entry does not exist with, %s = %s", key, value))
}

func (l *Logger) AssertLogEntryKeyExist(t assert.TestingT, key string) bool {
	for _, log := range l.recorded.All() {
		for _, r := range log.Context {
			if r.Key == key {
				return true
			}
		}
	}
	return assert.Fail(t, fmt.Sprintf("log entry does not exist with key = %s ", key))
}

const (
	traceIDKey  = "trace_id"
	spanIDKey   = "span_id"
	testTraceID = "test-trace-id-3333"
)

func TestSet(t *testing.T) {
	testLog := NewLogger(t)

	testTraceID2 := "test-trace-id-new"
	traceIDCtx := Set(context.Background(), []zap.Field{zap.String(traceIDKey, testTraceID)})
	tests := map[string]struct {
		context             context.Context
		expectedLoggerKey   string
		expectedLoggerValue string
	}{
		"context for zax filed is empty": {
			context:             Set(context.Background(), nil),
			expectedLoggerKey:   "",
			expectedLoggerValue: "",
		},
		"context with trace-id": {
			context:             traceIDCtx,
			expectedLoggerKey:   traceIDKey,
			expectedLoggerValue: testTraceID,
		},
		"context with trace-id with new value(to check it will be updated)": {
			context:             Set(traceIDCtx, []zap.Field{zap.String(traceIDKey, testTraceID2)}),
			expectedLoggerKey:   traceIDKey,
			expectedLoggerValue: testTraceID2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := tc.context
			logger := testLog.logger.With(GetAll(ctx)...)
			logger.Info("just a test record")
			assert.NotNil(t, logger)
			testLog.AssertLogEntryExist(t, tc.expectedLoggerKey, tc.expectedLoggerValue)
		})
	}
}

func TestAppend(t *testing.T) {
	testLog := NewLogger(t)
	ctx := context.Background()
	ctx = Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)})
	tests := map[string]struct {
		context             context.Context
		expectedFieldNumber int
	}{
		"context for zax filed is empty": {
			context:             Append(ctx, nil),
			expectedFieldNumber: 1,
		},
		"context with appending trace-id": {
			context:             Append(ctx, []zap.Field{zap.String(spanIDKey, testTraceID)}),
			expectedFieldNumber: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := tc.context
			logger := testLog.logger.With(GetAll(ctx)...)
			logger.Info("just a test record")
			assert.NotNil(t, logger)
			assert.Equal(t, tc.expectedFieldNumber, len(GetAll(ctx)))
		})
	}
}

func TestGet(t *testing.T) {
	testLog := NewLogger(t)
	traceIDKey := traceIDKey
	ctx := context.Background()
	tests := map[string]struct {
		context           context.Context
		expectedLoggerKey *string
	}{
		"context empty": {
			context:           context.TODO(),
			expectedLoggerKey: nil,
		},
		"context with trace-id field": {
			context:           Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)}),
			expectedLoggerKey: &traceIDKey,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := tc.context
			testLog.logger.With(GetAll(ctx)...).Info("just a test record")
			if tc.expectedLoggerKey != nil {
				testLog.AssertLogEntryKeyExist(t, *tc.expectedLoggerKey)
			}
		})
	}
}
func TestGetField(t *testing.T) {
	traceIDKey := traceIDKey
	ctx := context.Background()
	tests := map[string]struct {
		context       context.Context
		expectedOk    bool
		expectedValue string
	}{
		"context empty": {
			context:       context.TODO(),
			expectedOk:    false,
			expectedValue: "",
		},
		"context with trace-id field": {
			context:       Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)}),
			expectedOk:    true,
			expectedValue: testTraceID,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := tc.context
			field, ok := GetField(ctx, traceIDKey)
			assert.Equal(t, tc.expectedOk, ok)
			if ok {
				assert.Equal(t, tc.expectedValue, field.String)
			}
		})
	}
}

func TestGetFields(t *testing.T) {
	additionalKey := "additionalKey"
	ctx := Set(
		context.Background(), []zap.Field{
			zap.String(traceIDKey, testTraceID),
			zap.String(additionalKey, "additionalValue"),
			zap.String("superfluous", ""),
		},
	)
	absentKey := "absentKey"

	fields := GetFields(ctx, traceIDKey, additionalKey, absentKey)
	assert.Len(t, fields, 3)

	var absentKeysField zap.Field
	fieldKeys := make([]string, len(fields))
	for i, field := range fields {
		fieldKeys[i] = field.Key
		if field.Key == AbsentFieldsKey {
			absentKeysField = field
		}
	}

	assert.ElementsMatch(t, []string{traceIDKey, additionalKey, AbsentFieldsKey}, fieldKeys)
	t.Logf("%T %+v", absentKeysField.Interface, absentKeysField.Interface)

	assert.Equal(t, zapcore.ArrayMarshalerType, absentKeysField.Type)
	// BODGE: brittle to default print representation changes.
	assert.Equal(t, fmt.Sprintf("[%s]", absentKey), fmt.Sprint(absentKeysField.Interface))
}
