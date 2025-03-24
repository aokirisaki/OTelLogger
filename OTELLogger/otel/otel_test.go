package otel_test

import (
	"otellogger/otel"
	"otellogger/utils"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTransactionLog(t *testing.T) {
	attrs := map[string]string{"test": "test"}
	tlog := otel.NewTransactionLog(utils.LoggerName, utils.ServiceName, attrs)

	_, err := strconv.ParseInt(tlog.TraceID, 10, 64)
	assert.Equal(t, nil, err)
	assert.Equal(t, attrs, tlog.Attributes)
}

func TestNewOTelLog(t *testing.T) {
	attrs := map[string]string{"test": "test"}
	log := otel.NewOTelLog(utils.LoggerName, "1234567890", utils.ServiceName, "10.10.2025 17:00:00", "INFO", "message", attrs)

	_, err := strconv.ParseInt(log.SpanID, 10, 64)
	assert.Equal(t, nil, err)
	assert.Equal(t, "10.10.2025 17:00:00", log.Timestamp)
	assert.Equal(t, "INFO", log.Severity)
	assert.Equal(t, "message", log.Message)
	assert.Equal(t, utils.LoggerName, log.LoggerName)
	assert.Equal(t, utils.ServiceName, log.ServiceName)
	assert.Equal(t, "1234567890", log.TraceID)
	assert.Equal(t, attrs, log.Attributes)
}
