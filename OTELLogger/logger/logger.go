package logger

import (
	"encoding/json"
	"errors"
	"os"
	"otellogger/logExporter"
	"otellogger/otel"
	"otellogger/utils"
	"sync"
	"time"
)

// driver interface
type LogExporter interface {
	ExportLogs(traceID string, logs []*otel.OTelLog, config map[string]string) error
}

type Logger struct {
	LoggerName      string
	ServiceName     string
	mu              sync.Mutex
	Level           Level
	LogExporter     LogExporter
	TransactionLogs map[string]*otel.TransactionLog // mapped with key as trace ID
	config          map[string]string
}

type Level int

const (
	DEBUG   Level = 1
	INFO    Level = 2
	WARNING Level = 3
	ERROR   Level = 4
)

// create new logger with default logger name, service name and log exporter
func NewLogger(logLevel Level) *Logger {
	return &Logger{
		LoggerName:      utils.LoggerName,
		ServiceName:     utils.ServiceName,
		TransactionLogs: make(map[string]*otel.TransactionLog),
		LogExporter:     &logExporter.DefaultExporter{},
		Level:           logLevel,
	}
}

// configure the logger via a config file
func (l *Logger) WithConfig(filepath string) (*Logger, error) {
	l.config = make(map[string]string)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return l, err
	}

	err = json.Unmarshal(data, &l.config)
	if err != nil {
		return l, err
	}

	if len(l.config) != 0 {
		// check if there's a logger name provided and set it to that value
		loggerName, ok := l.config["loggerName"]
		if ok {
			l.LoggerName = loggerName
		}

		// check if there's a service name provided and set it to that value
		serviceName, ok := l.config["serviceName"]
		if ok {
			l.ServiceName = serviceName
		}

		// check if there's a log level provided and set it to that value
		level, ok := l.config["level"]
		if ok {
			switch level {
			case "INFO":
				l.Level = INFO
			case "DEBUG":
				l.Level = DEBUG
			case "WARNING":
				l.Level = WARNING
			case "ERROR":
				l.Level = ERROR
			default:
				l.Level = INFO
			}

		}
	}

	return l, nil
}

// give a custom exporter driver to the logger
func (l *Logger) WithExporter(exp LogExporter) *Logger {
	l.LogExporter = exp

	return l
}

// start logging for a transaction and return its trace ID
func (l *Logger) StartTransaction(attributes map[string]string) string {
	// lock the map
	l.mu.Lock()
	defer l.mu.Unlock()

	// create a new transaction log and add it to the map of transaction logs
	newTransaction := otel.NewTransactionLog(l.LoggerName, l.ServiceName, attributes)
	l.TransactionLogs[newTransaction.TraceID] = newTransaction

	return newTransaction.TraceID
}

func (l *Logger) SetLoggerName(name string) {
	l.LoggerName = name
}

func (l *Logger) SetServiceName(name string) {
	l.ServiceName = name
}

func (l *Logger) SetLevel(level Level) {
	l.Level = level
}

func (l *Logger) getLevel(level Level) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN LEVEL"
	}
}

// create log and add it to the corresponding transaction log
func (l *Logger) createLog(level Level, traceID, message string, attrs map[string]string) error {
	// check if the level is one that will show
	if level >= l.Level {
		l.mu.Lock()
		defer l.mu.Unlock()

		timestamp := time.Now().Format("02.01.2006 15:04:05")

		// check if the transaction log exists
		_, ok := l.TransactionLogs[traceID]
		if !ok {
			return errors.New("invalid trace ID")
		}

		// create the new log and add it to the transaction log
		lvl := l.getLevel(level)
		if lvl == "UNKNOWN LEVEL" {
			return errors.New("unknown log level")
		}

		otelLog := otel.NewOTelLog(l.LoggerName, traceID, l.ServiceName, timestamp, l.getLevel(level), message, attrs)
		l.TransactionLogs[traceID].Spans = append(l.TransactionLogs[traceID].Spans, otelLog)
	}

	return nil
}

func (l *Logger) Debug(message, traceID string, attrs map[string]string) error {
	return l.createLog(DEBUG, traceID, message, attrs)
}

func (l *Logger) Info(message, traceID string, attrs map[string]string) error {
	return l.createLog(INFO, traceID, message, attrs)
}

func (l *Logger) Warning(message, traceID string, attrs map[string]string) error {
	return l.createLog(WARNING, traceID, message, attrs)
}

func (l *Logger) Error(message, traceID string, attrs map[string]string) error {
	return l.createLog(ERROR, traceID, message, attrs)
}

// export logs for a transaction
func (l *Logger) ExportLogs(traceID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	transactionLog, ok := l.TransactionLogs[traceID]
	if !ok {
		return errors.New("invalid trace ID")
	}

	err := l.LogExporter.ExportLogs(transactionLog.TraceID, transactionLog.Spans, l.config)
	if err != nil {
		return err
	}

	// remove transaction log from map
	delete(l.TransactionLogs, traceID)

	return nil
}

// export all logs from all transactions
func (l *Logger) ExportAllLogs() error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(l.TransactionLogs))

	// export each transaction log on a separate goroutine
	for _, tLog := range l.TransactionLogs {
		wg.Add(1)

		go func(tLog *otel.TransactionLog) {
			defer wg.Done()

			err := l.ExportLogs(tLog.TraceID)
			if err != nil {
				errChan <- err
				return
			}
		}(tLog)
	}

	wg.Wait()

	close(errChan)

	// return the first error encountered if error
	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}
