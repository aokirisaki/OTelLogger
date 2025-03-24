package logExporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"otellogger/otel"
)

// the provided exporter drivers
// more can be added out of the box
type DefaultExporter struct{}
type JSONExporter struct{}
type TXTExporter struct{}

func parse(log *otel.OTelLog) ([]byte, error) {
	// marshal the map to json to get the desired format
	parsedLog, err := json.Marshal(log)
	if err != nil {
		return []byte{}, err
	}

	return parsedLog, nil
}

// default exporter is to console
func (exp *DefaultExporter) ExportLogs(traceID string, logs []*otel.OTelLog, config map[string]string) error {
	// iterate through the logs from a transaction and print them to console
	for _, log := range logs {
		parsedLog, err := parse(log)
		if err != nil {
			return err
		}

		fmt.Printf("[%s] [%s] %s\n", log.Severity, log.Timestamp, parsedLog)
	}
	return nil
}

// export logs as jsons
func (exp *JSONExporter) ExportLogs(traceID string, logs []*otel.OTelLog, config map[string]string) error {
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

	// all logfiles will have the format filename_1234567890.json to be able to recognize it by traceID
	file, err := os.Create(filepath + filename + "_" + traceID + ".json")
	if err != nil {
		return err
	}
	defer file.Close()

	// write the logs from a transaction to json file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(logs)
	if err != nil {
		return err
	}

	return nil
}

// export logs as text files
func (exp *TXTExporter) ExportLogs(traceID string, logs []*otel.OTelLog, config map[string]string) error {
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

	file, err := os.OpenFile(filepath+filename+"_"+traceID+".txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// write logs to text file
	for _, log := range logs {
		parsedLog, err := parse(log)
		if err != nil {
			return err
		}

		content := fmt.Sprintf("[%s] [%s] %s\n", log.Severity, log.Timestamp, parsedLog)

		_, err = file.WriteString(content)
		if err != nil {
			return err
		}
	}
	return nil
}
