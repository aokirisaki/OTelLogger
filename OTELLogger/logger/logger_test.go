package logger_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"otellogger/logExporter"
	"otellogger/logger"
	"otellogger/otel"
	"otellogger/utils"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mock struct for exporter
type MockExporter struct {
	mock.Mock
}

// mock for ExportLogs function
func (m *MockExporter) ExportLogs(traceID string, logs []*otel.OTelLog, config map[string]string) error {
	return errors.New("mock error")
}

const CUSTOM = "TEST Severity level: INFO Message: info message key1=val1 \n" +
	"TEST Severity level: DEBUG Message: debug message key2=val2 \n"

// out of the box exporter
type TestExporter struct{}

func (c *TestExporter) ExportLogs(traceID string, logs []*otel.OTelLog, config map[string]string) error {
	// check if there are no logs to export
	if len(logs) == 0 {
		return nil
	}

	if config == nil {
		return errors.New("no config provided")
	}

	// get the filepath from config
	filepath, ok := config["filepath"]
	if !ok {
		return errors.New("no filepath in config")
	}

	// get the way the filename will look like
	filename, ok := config["filename"]
	if !ok {
		return errors.New("no filename in config")
	}

	file, err := os.OpenFile(filepath+filename+"_custom.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// write logs to text file
	for _, log := range logs {
		attrs := ""
		for attr, val := range log.Attributes {
			attrs += attr + "=" + val + " "
		}
		content := fmt.Sprintf("TEST Severity level: %s Message: %s %s\n", log.Severity, log.Message, attrs)

		_, err = file.WriteString(content)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestNewLogger(t *testing.T) {
	l := logger.NewLogger(logger.INFO)

	assert.Equal(t, utils.LoggerName, l.LoggerName)
	assert.Equal(t, utils.ServiceName, l.ServiceName)
	assert.Equal(t, logger.INFO, l.Level)
	assert.Equal(t, reflect.TypeOf(&logExporter.DefaultExporter{}), reflect.TypeOf(l.LogExporter))
}

func TestWithConfig(t *testing.T) {
	t.Run("create new logger with config successful", TestWithConfig_Success)
	t.Run("could not create new logger with config - invalid config path", TestWithConfig_ErrorInvalidCfgPath)
	t.Run("could not create new logger with config - invalid config format", TestWithConfig_ErrorInvalidCfgFormat)
}

func TestWithConfig_Success(t *testing.T) {
	// create config file
	cfg, err := os.Create("test_config.json")
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	config := struct {
		Filepath    string `json:"filepath"`
		Filename    string `json:"filename"`
		LoggerName  string `json:"loggerName"`
		ServiceName string `json:"serviceName"`
		Level       string `json:"level"`
	}{
		Filepath:    "",
		Filename:    "test_success",
		LoggerName:  "Test",
		ServiceName: "TestService",
		Level:       "DEBUG",
	}

	encoder := json.NewEncoder(cfg)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(config)
	if err != nil {
		t.Fatalf("Error writing config file: %v", err)
	}
	cfg.Close()

	// create logger with config
	l, err := logger.NewLogger(logger.INFO).WithConfig("test_config.json")

	assert.Equal(t, nil, err)
	assert.Equal(t, "Test", l.LoggerName)
	assert.Equal(t, "TestService", l.ServiceName)
	assert.Equal(t, logger.DEBUG, l.Level)
	assert.Equal(t, reflect.TypeOf(&logExporter.DefaultExporter{}), reflect.TypeOf(l.LogExporter))

	// remove config file
	err = os.Remove("test_config.json")
	if err != nil {
		t.Fatalf("Error removing config file: %v", err)
	}
}

func TestWithConfig_ErrorInvalidCfgPath(t *testing.T) {
	_, err := logger.NewLogger(logger.INFO).WithConfig("test_config_invalid.json")

	assert.NotEqual(t, nil, err)
	assert.Equal(t, "open test_config_invalid.json: The system cannot find the file specified.", err.Error())
}

func TestWithConfig_ErrorInvalidCfgFormat(t *testing.T) {
	// create config file
	cfg, err := os.Create("test_config_invalid_format.json")
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	config := struct {
		IntVal int `json:"intval"`
	}{
		IntVal: 123,
	}

	encoder := json.NewEncoder(cfg)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(config)
	if err != nil {
		t.Fatalf("Error writing config file: %v", err)
	}
	cfg.Close()

	_, err = logger.NewLogger(logger.INFO).WithConfig("test_config_invalid_format.json")

	assert.NotEqual(t, nil, err)
	assert.Equal(t, "json: cannot unmarshal number into Go value of type string", err.Error())

	err = os.Remove("test_config_invalid_format.json")
	if err != nil {
		t.Fatalf("Error removing config file: %v", err)
	}
}

func TestWithExporter(t *testing.T) {
	// create logger eith out of the box exporter
	l := logger.NewLogger(logger.INFO).WithExporter(&logExporter.JSONExporter{})

	assert.Equal(t, reflect.TypeOf(&logExporter.JSONExporter{}), reflect.TypeOf(l.LogExporter))
}

func TestStartTransaction(t *testing.T) {
	l := logger.NewLogger(logger.INFO)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	_, err := strconv.ParseInt(traceID, 10, 64)
	assert.Equal(t, nil, err)
}

func TestSetLoggerName(t *testing.T) {
	l := logger.NewLogger(logger.INFO)

	l.SetLoggerName("Test")
	assert.Equal(t, "Test", l.LoggerName)
}

func TestSetServiceName(t *testing.T) {
	l := logger.NewLogger(logger.INFO)

	l.SetServiceName("Test")
	assert.Equal(t, "Test", l.ServiceName)
}

func TestSetLevel(t *testing.T) {
	l := logger.NewLogger(logger.INFO)

	l.SetLevel(logger.DEBUG)
	assert.Equal(t, logger.DEBUG, l.Level)
}

func TestDebug(t *testing.T) {
	t.Run("Create logs for debug level successful", TestDebug_Success)
	t.Run("Error creating log for debug level", TestDebug_Error)
}

func TestDebug_Success(t *testing.T) {
	l := logger.NewLogger(logger.DEBUG)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	// generate logs - debug level will contain all other levels
	err := l.Debug("debug log", traceID, map[string]string{"key1": "val1"})
	assert.Equal(t, nil, err)

	err = l.Info("info log", traceID, map[string]string{"key2": "val2"})
	assert.Equal(t, nil, err)

	err = l.Warning("warning log", traceID, map[string]string{"key3": "val3"})
	assert.Equal(t, nil, err)

	err = l.Error("error log", traceID, map[string]string{"key4": "val4"})
	assert.Equal(t, nil, err)

	// check if all logging generated logs (debug should generate for all levels)
	assert.Equal(t, 4, len(l.TransactionLogs[traceID].Spans))
	// check their info
	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[0].TraceID)
	assert.Equal(t, "debug log", l.TransactionLogs[traceID].Spans[0].Message)
	assert.Equal(t, map[string]string{"key1": "val1"}, l.TransactionLogs[traceID].Spans[0].Attributes)

	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[1].TraceID)
	assert.Equal(t, "info log", l.TransactionLogs[traceID].Spans[1].Message)
	assert.Equal(t, map[string]string{"key2": "val2"}, l.TransactionLogs[traceID].Spans[1].Attributes)

	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[2].TraceID)
	assert.Equal(t, "warning log", l.TransactionLogs[traceID].Spans[2].Message)
	assert.Equal(t, map[string]string{"key3": "val3"}, l.TransactionLogs[traceID].Spans[2].Attributes)

	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[3].TraceID)
	assert.Equal(t, "error log", l.TransactionLogs[traceID].Spans[3].Message)
	assert.Equal(t, map[string]string{"key4": "val4"}, l.TransactionLogs[traceID].Spans[3].Attributes)
}

func TestDebug_Error(t *testing.T) {
	l := logger.NewLogger(logger.DEBUG)

	err := l.Debug("debug log", "invalid trace ID", map[string]string{"key1": "val1"})
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "invalid trace ID", err.Error())
}

func TestInfo(t *testing.T) {
	t.Run("Create logs for info level successful", TestInfo_Success)
	t.Run("Error creating log for debug level", TestInfo_Error)
}

func TestInfo_Success(t *testing.T) {
	l := logger.NewLogger(logger.INFO)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	// generate logs - info level will contain info, warning and error levels
	err := l.Debug("debug log", traceID, map[string]string{"key1": "val1"})
	assert.Equal(t, nil, err)

	err = l.Info("info log", traceID, map[string]string{"key2": "val2"})
	assert.Equal(t, nil, err)

	err = l.Warning("warning log", traceID, map[string]string{"key3": "val3"})
	assert.Equal(t, nil, err)

	err = l.Error("error log", traceID, map[string]string{"key4": "val4"})
	assert.Equal(t, nil, err)

	// check if all logging generated logs (debug should generate for all levels)
	assert.Equal(t, 3, len(l.TransactionLogs[traceID].Spans))

	// check their info
	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[0].TraceID)
	assert.Equal(t, "info log", l.TransactionLogs[traceID].Spans[0].Message)
	assert.Equal(t, map[string]string{"key2": "val2"}, l.TransactionLogs[traceID].Spans[0].Attributes)

	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[1].TraceID)
	assert.Equal(t, "warning log", l.TransactionLogs[traceID].Spans[1].Message)
	assert.Equal(t, map[string]string{"key3": "val3"}, l.TransactionLogs[traceID].Spans[1].Attributes)

	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[2].TraceID)
	assert.Equal(t, "error log", l.TransactionLogs[traceID].Spans[2].Message)
	assert.Equal(t, map[string]string{"key4": "val4"}, l.TransactionLogs[traceID].Spans[2].Attributes)
}

func TestInfo_Error(t *testing.T) {
	l := logger.NewLogger(logger.INFO)

	err := l.Info("info log", "invalid trace ID", map[string]string{"key1": "val1"})
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "invalid trace ID", err.Error())
}

func TestWarning(t *testing.T) {
	t.Run("Create logs for warning level successful", TestWarning_Success)
	t.Run("Error creating log for warning level", TestWarning_Error)
}

func TestWarning_Success(t *testing.T) {
	l := logger.NewLogger(logger.WARNING)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	// generate logs - warning level will contain warning and error levels
	err := l.Debug("debug log", traceID, map[string]string{"key1": "val1"})
	assert.Equal(t, nil, err)

	err = l.Info("info log", traceID, map[string]string{"key2": "val2"})
	assert.Equal(t, nil, err)

	err = l.Warning("warning log", traceID, map[string]string{"key3": "val3"})
	assert.Equal(t, nil, err)

	err = l.Error("error log", traceID, map[string]string{"key4": "val4"})
	assert.Equal(t, nil, err)

	// check if all logging generated logs (warning should generate for warning and error)
	assert.Equal(t, 2, len(l.TransactionLogs[traceID].Spans))

	// check their info
	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[0].TraceID)
	assert.Equal(t, "warning log", l.TransactionLogs[traceID].Spans[0].Message)
	assert.Equal(t, map[string]string{"key3": "val3"}, l.TransactionLogs[traceID].Spans[0].Attributes)

	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[1].TraceID)
	assert.Equal(t, "error log", l.TransactionLogs[traceID].Spans[1].Message)
	assert.Equal(t, map[string]string{"key4": "val4"}, l.TransactionLogs[traceID].Spans[1].Attributes)
}

func TestWarning_Error(t *testing.T) {
	l := logger.NewLogger(logger.WARNING)

	err := l.Warning("warning log", "invalid trace ID", map[string]string{"key1": "val1"})
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "invalid trace ID", err.Error())
}

func TestError(t *testing.T) {
	t.Run("Create logs for error level successful", TestError_Success)
	t.Run("Error creating log for error level", TestError_Error)
}

func TestError_Success(t *testing.T) {
	l := logger.NewLogger(logger.ERROR)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	// generate logs - error level will contain only error
	err := l.Debug("debug log", traceID, map[string]string{"key1": "val1"})
	assert.Equal(t, nil, err)

	err = l.Info("info log", traceID, map[string]string{"key2": "val2"})
	assert.Equal(t, nil, err)

	err = l.Warning("warning log", traceID, map[string]string{"key3": "val3"})
	assert.Equal(t, nil, err)

	err = l.Error("error log", traceID, map[string]string{"key4": "val4"})
	assert.Equal(t, nil, err)

	// check if all logging generated logs (error should generate only for error level)
	assert.Equal(t, 1, len(l.TransactionLogs[traceID].Spans))

	// check their info
	assert.Equal(t, traceID, l.TransactionLogs[traceID].Spans[0].TraceID)
	assert.Equal(t, "error log", l.TransactionLogs[traceID].Spans[0].Message)
	assert.Equal(t, map[string]string{"key4": "val4"}, l.TransactionLogs[traceID].Spans[0].Attributes)
}

func TestError_Error(t *testing.T) {
	l := logger.NewLogger(logger.ERROR)

	err := l.Error("error log", "invalid trace ID", map[string]string{"key1": "val1"})
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "invalid trace ID", err.Error())
}

func TestCustomExporter(t *testing.T) {
	// create config file
	cfg, err := os.Create("test_custom_config.json")
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	config := struct {
		Filepath string `json:"filepath"`
		Filename string `json:"filename"`
	}{
		Filepath: "",
		Filename: "test",
	}

	encoder := json.NewEncoder(cfg)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(config)
	if err != nil {
		t.Fatalf("Error writing config file: %v", err)
	}
	cfg.Close()

	// create logger with a custom config, out of the box
	l, err := logger.NewLogger(logger.DEBUG).WithConfig("test_custom_config.json")
	assert.Equal(t, nil, err)
	l = l.WithExporter(&TestExporter{})

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	// generate logs
	err = l.Info("info message", traceID, map[string]string{"key1": "val1"})
	assert.Equal(t, nil, err)

	err = l.Debug("debug message", traceID, map[string]string{"key2": "val2"})
	assert.Equal(t, nil, err)

	// export logs
	err = l.ExportLogs(traceID)
	assert.Equal(t, nil, err)

	// check the contents of the file with the exported logs
	content, err := os.ReadFile("test_custom.txt")
	if err != nil {
		t.Fatalf("Error reading logfile: %v", err)
	}

	assert.Equal(t, CUSTOM, string(content))

	// remove config file
	err = os.Remove("test_custom_config.json")
	if err != nil {
		t.Fatalf("Error removing config file: %v", err)
	}

	// remove test logfile
	err = os.Remove("test_custom.txt")
	if err != nil {
		t.Fatalf("Error removing logfile: %v", err)
	}
}

func TestExportLogs(t *testing.T) {
	t.Run("Export logs successful", TestExportLogs_Success)
	t.Run("Error exporting logs - invalid trace ID", TestExportLogs_ErrorInvalidTraceID)
	t.Run("Error exporting logs - log exporter returns error", TestExportLogs_ErrorOnLogExporter)
}

func TestExportLogs_Success(t *testing.T) {
	var buf bytes.Buffer
	originalStdout := os.Stdout

	// redirect stdout to buffer
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	l := logger.NewLogger(logger.DEBUG)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	// generate logs
	err = l.Debug("debug message", traceID, map[string]string{"test": "test"})
	assert.Equal(t, nil, err)

	// keep the transaction log for testing
	tlog := l.TransactionLogs[traceID]

	// export logs
	err = l.ExportLogs(traceID)
	assert.Equal(t, nil, err)

	// return to stdout
	w.Close()
	os.Stdout = originalStdout
	io.Copy(&buf, r)

	expected := `[DEBUG] [` + tlog.Spans[0].Timestamp + `] {"Timestamp":"` +
		tlog.Spans[0].Timestamp + `","Severity":"DEBUG","Message":"debug message",` +
		`"LoggerName":"OTelLogger","ServiceName":"Default","TraceID":"` + traceID + `","SpanID":"` +
		tlog.Spans[0].SpanID + `","Attributes":{"test":"test"}}` + "\n"

	assert.Equal(t, expected, buf.String())
}

func TestExportLogs_ErrorInvalidTraceID(t *testing.T) {
	l := logger.NewLogger(logger.DEBUG)

	err := l.ExportLogs("1234567890")
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "invalid trace ID", err.Error())
}

func TestExportLogs_ErrorOnLogExporter(t *testing.T) {
	l := logger.NewLogger(logger.DEBUG)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	err := l.Debug("debug message", traceID, map[string]string{"key": "val"})
	assert.Equal(t, nil, err)

	// create a mock exporter
	mockExporter := new(MockExporter)

	// set the expected behaviour for the mock exporter
	mockExporter.On(traceID, l.TransactionLogs[traceID].Spans, l.TransactionLogs[traceID].Attributes).Return(errors.New("mocked error"))

	// use the mock exporter that will return error when exporting logs
	l = l.WithExporter(mockExporter)

	// check for error
	err = l.ExportLogs(traceID)
	assert.Equal(t, "mock error", err.Error())
}

func TestExportAllLogs(t *testing.T) {
	t.Run("Export all logs successful", TestExportAllLogs_Success)
	t.Run("Error exporting all logs - log exporter returns error", TestExportAllLogs_ErrorOnLogExporter)
}

func TestExportAllLogs_Success(t *testing.T) {
	var buf bytes.Buffer
	originalStdout := os.Stdout

	// redirect stdout to buffer
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	l := logger.NewLogger(logger.DEBUG)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	// generate logs
	err = l.Debug("debug message", traceID, map[string]string{"key": "val"})
	assert.Equal(t, nil, err)

	// start new transaction
	traceID2 := l.StartTransaction(map[string]string{"test2": "test2"})

	// generate logs for the second transaction
	err = l.Info("info message", traceID2, map[string]string{"key2": "val2"})
	assert.Equal(t, nil, err)

	// keep the transaction logs for testing
	tlogs := make(map[string]*otel.TransactionLog)
	for key, val := range l.TransactionLogs {
		tlogs[key] = val
	}

	// export logs
	err = l.ExportAllLogs()
	assert.Equal(t, nil, err)

	// return to stdout
	w.Close()
	os.Stdout = originalStdout
	io.Copy(&buf, r)

	transaction1 := `[DEBUG] [` + tlogs[traceID].Spans[0].Timestamp + `] {"Timestamp":"` +
		tlogs[traceID].Spans[0].Timestamp + `","Severity":"DEBUG","Message":"debug message",` +
		`"LoggerName":"OTelLogger","ServiceName":"Default","TraceID":"` + traceID + `","SpanID":"` +
		tlogs[traceID].Spans[0].SpanID + `","Attributes":{"key":"val"}}` + "\n"

	transaction2 := `[INFO] [` + tlogs[traceID2].Spans[0].Timestamp + `] {"Timestamp":"` +
		tlogs[traceID2].Spans[0].Timestamp + `","Severity":"INFO","Message":"info message",` +
		`"LoggerName":"OTelLogger","ServiceName":"Default","TraceID":"` + traceID2 + `","SpanID":"` +
		tlogs[traceID2].Spans[0].SpanID + `","Attributes":{"key2":"val2"}}` + "\n"

	// since logs will be exported in any order (goroutines) we need to check for both cases of output
	assert.True(t, transaction1+transaction2 == buf.String() || transaction2+transaction1 == buf.String())
}

func TestExportAllLogs_ErrorOnLogExporter(t *testing.T) {
	l := logger.NewLogger(logger.DEBUG)

	traceID := l.StartTransaction(map[string]string{"test": "test"})

	err := l.Debug("debug message", traceID, map[string]string{"key": "val"})
	assert.Equal(t, nil, err)

	// create a mock exporter
	mockExporter := new(MockExporter)

	// set the expected behaviour for the mock exporter
	mockExporter.On(traceID, l.TransactionLogs[traceID].Spans, l.TransactionLogs[traceID].Attributes).Return(errors.New("mocked error"))

	// use the mock exporter that will return error when exporting logs
	l = l.WithExporter(mockExporter)

	// check for error
	err = l.ExportAllLogs()
	assert.Equal(t, "mock error", err.Error())
}

func TestLoggingFromMultipleGoroutines(t *testing.T) {
	// create config file
	cfg, err := os.Create("test_config.json")
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	config := struct {
		Filepath string `json:"filepath"`
		Filename string `json:"filename"`
	}{
		Filepath: "",
		Filename: "goroutine_log",
	}

	encoder := json.NewEncoder(cfg)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(config)
	if err != nil {
		t.Fatalf("Error writing config file: %v", err)
	}
	cfg.Close()

	l, err := logger.NewLogger(logger.INFO).WithConfig("test_config.json")
	assert.Equal(t, nil, err)
	l = l.WithExporter(&logExporter.TXTExporter{})

	err = os.Remove("test_config.json")
	if err != nil {
		t.Fatalf("Error removing config file: %v", err)
	}

	// start each transaction on a separate goroutine
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			traceID := l.StartTransaction(map[string]string{"test": "test"})

			err := l.Info("info message", traceID, map[string]string{"key1": "val1"})
			assert.Equal(t, nil, err)

			err = l.Warning("warning message", traceID, map[string]string{"key2": "val2"})
			assert.Equal(t, nil, err)

			// keep logs slice for testing
			logs := l.TransactionLogs[traceID].Spans

			err = l.ExportLogs(traceID)
			assert.Equal(t, nil, err)

			// check if logs have been successfully exported for each transaction
			expected := `[INFO] [` + logs[0].Timestamp + `] {"Timestamp":"` +
				logs[0].Timestamp + `","Severity":"INFO","Message":"info message",` +
				`"LoggerName":"OTelLogger","ServiceName":"Default","TraceID":"` + traceID + `","SpanID":"` +
				logs[0].SpanID + `","Attributes":{"key1":"val1"}}` + "\n" +
				`[WARNING] [` + logs[1].Timestamp + `] {"Timestamp":"` +
				logs[1].Timestamp + `","Severity":"WARNING","Message":"warning message",` +
				`"LoggerName":"OTelLogger","ServiceName":"Default","TraceID":"` + traceID + `","SpanID":"` +
				logs[1].SpanID + `","Attributes":{"key2":"val2"}}` + "\n"

			content, err := os.ReadFile("goroutine_log_" + traceID + ".txt")
			assert.Equal(t, nil, err)
			assert.Equal(t, expected, string(content))

			err = os.Remove("goroutine_log_" + traceID + ".txt")
			assert.Equal(t, nil, err)
		}()
	}

	wg.Wait()
}
