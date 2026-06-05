package lib

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	ll "github.com/oresoftware/json-logging/jlog/level"
)

func decodeJSONLines(t *testing.T, raw []byte) [][]interface{} {
	t.Helper()

	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	records := make([][]interface{}, 0, len(lines))

	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		if line[0] != '[' {
			t.Fatalf("expected JSON array record, got %q", line[0])
		}

		var record []interface{}
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("could not decode JSON log record %q: %v", string(line), err)
		}

		if len(record) != 8 {
			t.Fatalf("expected 8 fields in JSON log record, got %d: %#v", len(record), record)
		}

		records = append(records, record)
	}

	return records
}

func tempLogFile(t *testing.T) *os.File {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "jlog-*.log")
	if err != nil {
		t.Fatal(err)
	}

	return f
}

func readLogFile(t *testing.T, f *os.File) []byte {
	t.Helper()

	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	raw, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	return raw
}

func TestJSONStdoutUsesArrayFormat(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	oldStdout := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	log := CreateLogger("stdio-json").SetToJSONOutput().SetLogLevel(ll.TRACE)
	log.Info(MP("requestId", "req-1"), "hello", 42)

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = oldStdout

	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	records := decodeJSONLines(t, raw)
	if len(records) != 1 {
		t.Fatalf("expected one JSON record, got %d", len(records))
	}

	record := records[0]
	if record[0] != "@bunion:v1" {
		t.Fatalf("unexpected marker: %#v", record[0])
	}
	if record[1] != "stdio-json" {
		t.Fatalf("unexpected app name: %#v", record[1])
	}
	if record[2] != "INFO" {
		t.Fatalf("unexpected level: %#v", record[2])
	}

	meta, ok := record[6].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata object, got %T", record[6])
	}
	if meta["requestId"] != "req-1" {
		t.Fatalf("expected requestId metadata, got %#v", meta)
	}
	if _, ok := meta["log_num"]; !ok {
		t.Fatalf("expected log_num metadata, got %#v", meta)
	}

	messages, ok := record[7].([]interface{})
	if !ok {
		t.Fatalf("expected message array, got %T", record[7])
	}
	if len(messages) != 2 || messages[0] != "hello" || messages[1] != float64(42) {
		t.Fatalf("unexpected messages: %#v", messages)
	}
}

func TestJSONLevelGuardsSkipDisabledLogs(t *testing.T) {
	f := tempLogFile(t)

	log := CreateLogger("guard-json").
		SetOutputFile(f).
		SetToJSONOutput().
		SetLogLevel(ll.WARN)

	if log.V(ll.DEBUG) {
		t.Fatal("debug should be disabled at WARN")
	}
	if log.IsInfoEnabled() {
		t.Fatal("info should be disabled at WARN")
	}
	if !log.IsWarnEnabled() {
		t.Fatal("warn should be enabled at WARN")
	}

	log.Debug("hidden debug")
	log.Info("hidden info")
	log.Warn("visible warn")

	records := decodeJSONLines(t, readLogFile(t, f))
	if len(records) != 1 {
		t.Fatalf("expected one enabled record, got %d", len(records))
	}
	if records[0][2] != "WARN" {
		t.Fatalf("unexpected level: %#v", records[0][2])
	}
	if messages := records[0][7].([]interface{}); messages[0] != "visible warn" {
		t.Fatalf("unexpected message payload: %#v", messages)
	}
}

func TestJSONCircularFallbackStaysValid(t *testing.T) {
	type node struct {
		Name string
		Next *node
	}

	logFile := tempLogFile(t)
	warnFile := tempLogFile(t)

	DefaultLogger.Mtx.Lock()
	oldDefaultFile := DefaultLogger.File
	DefaultLogger.File = warnFile
	DefaultLogger.Mtx.Unlock()
	t.Cleanup(func() {
		DefaultLogger.Mtx.Lock()
		DefaultLogger.File = oldDefaultFile
		DefaultLogger.Mtx.Unlock()
	})

	n := &node{Name: "root"}
	n.Next = n

	log := CreateLogger("circular-json").
		SetOutputFile(logFile).
		SetToJSONOutput().
		SetLogLevel(ll.TRACE)

	log.Info(n)

	raw := readLogFile(t, logFile)
	records := decodeJSONLines(t, raw)
	if len(records) != 1 {
		t.Fatalf("expected one circular fallback record, got %d", len(records))
	}
	if !strings.Contains(string(raw), "(go:circular:") {
		t.Fatalf("expected circular reference marker in output: %s", string(raw))
	}
}
