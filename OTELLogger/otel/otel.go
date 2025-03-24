package otel

import (
	"math/rand"
	"strconv"
)

// log structure
type OTelLog struct {
	Timestamp   string            `json:"Timestamp"`
	Severity    string            `json:"Severity"`
	Message     string            `json:"Message"`
	LoggerName  string            `json:"LoggerName"`
	ServiceName string            `json:"ServiceName"`
	TraceID     string            `json:"TraceID"`
	SpanID      string            `json:"SpanID"`
	Attributes  map[string]string `json:"Attributes"`
}

// transaction-styled log (contains multiple OTelLogs)
type TransactionLog struct {
	TraceID    string
	Spans      []*OTelLog
	Attributes map[string]string
}

// create new transaction log and generate its trace ID
func NewTransactionLog(loggerName, serviceName string, attributes map[string]string) *TransactionLog {
	return &TransactionLog{
		TraceID:    strconv.FormatInt(rand.Int63(), 10),
		Attributes: attributes,
	}
}

// create new log
func NewOTelLog(loggerName, traceID, serviceName, timestamp, level, message string, attributes map[string]string) *OTelLog {
	return &OTelLog{
		Timestamp:   timestamp,
		SpanID:      strconv.FormatInt(rand.Int63(), 10),
		Severity:    level,
		Message:     message,
		LoggerName:  loggerName,
		TraceID:     traceID,
		ServiceName: serviceName,
		Attributes:  attributes,
	}
}
