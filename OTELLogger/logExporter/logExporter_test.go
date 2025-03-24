package logExporter_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"otellogger/logExporter"
	"otellogger/otel"
	"otellogger/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

const LOGS = `[INFO] [10.03.2025 17:00:00] {"Timestamp":"10.03.2025 17:00:00","Severity":"INFO",` +
	`"Message":"test message 1","LoggerName":"OTelLogger","ServiceName":"Default",` +
	`"TraceID":"1234567890","SpanID":"00000000000","Attributes":{"key1":"val1"}}` + "\n" +
	`[INFO] [10.03.2025 17:01:00] {"Timestamp":"10.03.2025 17:01:00","Severity":"INFO",` +
	`"Message":"test message 2","LoggerName":"OTelLogger","ServiceName":"Default",` +
	`"TraceID":"1234567890","SpanID":"00000000001","Attributes":{"key2":"val2"}}` + "\n"

// helper function for testing
func createTestLog() []*otel.OTelLog {
	return []*otel.OTelLog{
		{
			Timestamp:   "10.03.2025 17:00:00",
			Severity:    "INFO",
			Message:     "test message 1",
			LoggerName:  utils.LoggerName,
			ServiceName: utils.ServiceName,
			TraceID:     "1234567890",
			SpanID:      "00000000000",
			Attributes:  map[string]string{"key1": "val1"},
		},
		{
			Timestamp:   "10.03.2025 17:01:00",
			Severity:    "INFO",
			Message:     "test message 2",
			LoggerName:  utils.LoggerName,
			ServiceName: utils.ServiceName,
			TraceID:     "1234567890",
			SpanID:      "00000000001",
			Attributes:  map[string]string{"key2": "val2"},
		},
	}
}

func TestExportLogsDefault(t *testing.T) {
	// redirect stdout to buffer to be able to assert output for default exporter
	var buf bytes.Buffer
	originalStdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	defaultLogExporter := logExporter.DefaultExporter{}

	otellogs := createTestLog()
	err = defaultLogExporter.ExportLogs("1234567890", otellogs, nil)

	// return to stdout
	w.Close()
	os.Stdout = originalStdout
	io.Copy(&buf, r)

	assert.Equal(t, nil, err)
	assert.Equal(t, LOGS, buf.String())
}

func TestExportLogsJSON(t *testing.T) {
	t.Run("Export logs to json successful", TestExportLogsJSON_Success)
	t.Run("Error exporting logs to json - no config provided", TestExportLogsJSON_NoConfig)
	t.Run("Error exporting logs to json - no filepath in config", TestExportLogsJSON_NoFilepath)
	t.Run("Error exporting logs to json - no filename in config", TestExportLogsJSON_NoFilename)
}

func TestExportLogsJSON_Success(t *testing.T) {
	jsonLogExporter := logExporter.JSONExporter{}
	otellogs := createTestLog()

	err := jsonLogExporter.ExportLogs("1234567890", nil, nil)
	assert.Equal(t, nil, err)

	err = jsonLogExporter.ExportLogs("1234567890", otellogs, map[string]string{"filepath": "", "filename": "test_json_success"})
	assert.Equal(t, nil, err)

	// open the logfile
	file, err := os.Open("test_json_success_1234567890.json")
	// check if the logfile has been created successfully
	if err != nil {
		t.Fatalf("Error opening file: %v", err)
	}

	var log []*otel.OTelLog

	if err != nil {
		return
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&log)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	file.Close()

	assert.Equal(t, &otel.OTelLog{
		Timestamp:   "10.03.2025 17:00:00",
		Severity:    "INFO",
		Message:     "test message 1",
		LoggerName:  utils.LoggerName,
		ServiceName: utils.ServiceName,
		TraceID:     "1234567890",
		SpanID:      "00000000000",
		Attributes:  map[string]string{"key1": "val1"},
	}, log[0])

	assert.Equal(t, &otel.OTelLog{
		Timestamp:   "10.03.2025 17:01:00",
		Severity:    "INFO",
		Message:     "test message 2",
		LoggerName:  utils.LoggerName,
		ServiceName: utils.ServiceName,
		TraceID:     "1234567890",
		SpanID:      "00000000001",
		Attributes:  map[string]string{"key2": "val2"},
	}, log[1])

	// remove test file
	err = os.Remove("test_json_success_1234567890.json")
	if err != nil {
		t.Fatalf("Error removing file: %v", err)
	}
}

func TestExportLogsJSON_NoConfig(t *testing.T) {
	jsonLogExporter := logExporter.JSONExporter{}
	otellogs := createTestLog()

	err := jsonLogExporter.ExportLogs("1234567890", otellogs, nil)
	assert.Equal(t, errors.New("no config provided"), err)
}

func TestExportLogsJSON_NoFilepath(t *testing.T) {
	jsonLogExporter := logExporter.JSONExporter{}
	otellogs := createTestLog()

	err := jsonLogExporter.ExportLogs("1234567890", otellogs, map[string]string{"filename": "test_filepath"})
	assert.Equal(t, errors.New("no filepath in config"), err)
}

func TestExportLogsJSON_NoFilename(t *testing.T) {
	jsonLogExporter := logExporter.JSONExporter{}
	otellogs := createTestLog()

	err := jsonLogExporter.ExportLogs("1234567890", otellogs, map[string]string{"filepath": ""})
	assert.Equal(t, errors.New("no filename in config"), err)
}

func TestExportLogsTxt(t *testing.T) {
	t.Run("Export logs to text file successful", TestExportLogsTxt_Success)
	t.Run("Error exporting logs to text file - no config provided", TestExportLogsTxt_NoConfig)
	t.Run("Error exporting logs to text file - no filepath in config", TestExportLogsTxt_NoFilepath)
	t.Run("Error exporting logs to text file - no filename in config", TestExportLogsTxt_NoFilename)
}

func TestExportLogsTxt_Success(t *testing.T) {
	txtLogExporter := logExporter.TXTExporter{}
	otellogs := createTestLog()

	err := txtLogExporter.ExportLogs("1234567890", nil, nil)
	assert.Equal(t, nil, err)

	err = txtLogExporter.ExportLogs("1234567890", otellogs, map[string]string{"filepath": "", "filename": "test_txt_success"})
	assert.Equal(t, nil, err)

	content, err := os.ReadFile("test_txt_success_1234567890.txt")
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}

	assert.Equal(t, LOGS, string(content))

	// remove test file
	err = os.Remove("test_txt_success_1234567890.txt")
	if err != nil {
		t.Fatalf("Error removing file: %v", err)
	}
}

func TestExportLogsTxt_NoConfig(t *testing.T) {
	txtLogExporter := logExporter.TXTExporter{}
	otellogs := createTestLog()

	err := txtLogExporter.ExportLogs("1234567890", otellogs, nil)
	assert.Equal(t, errors.New("no config provided"), err)
}

func TestExportLogsTxt_NoFilepath(t *testing.T) {
	txtLogExporter := logExporter.TXTExporter{}
	otellogs := createTestLog()

	err := txtLogExporter.ExportLogs("1234567890", otellogs, map[string]string{"filename": "test_filepath"})
	assert.Equal(t, errors.New("no filepath in config"), err)
}

func TestExportLogsTxt_NoFilename(t *testing.T) {
	txtLogExporter := logExporter.TXTExporter{}
	otellogs := createTestLog()

	err := txtLogExporter.ExportLogs("1234567890", otellogs, map[string]string{"filepath": ""})
	assert.Equal(t, errors.New("no filename in config"), err)
}
