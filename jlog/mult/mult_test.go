package mult

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
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

	f, err := os.CreateTemp(t.TempDir(), "jlog-mult-*.log")
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

func TestMultiLoggerJSONArrayFormatAndFileLevels(t *testing.T) {
	warnFile := tempLogFile(t)
	infoFile := tempLogFile(t)

	log := New("multi-json", "", []*FileLevel{{
		Level:  ll.WARN,
		File:   warnFile,
		IsJSON: true,
	}})
	log.AddOutputFile(ll.INFO, infoFile).SetToJSONOutput()

	if log.V(ll.DEBUG) {
		t.Fatal("debug should be disabled with WARN and INFO outputs")
	}
	if !log.IsInfoEnabled() {
		t.Fatal("info should be enabled by the INFO output")
	}
	if !log.IsWarnEnabled() {
		t.Fatal("warn should be enabled by both outputs")
	}

	log.Debug("hidden debug")
	log.Info("visible info")
	log.Warn("visible warn")

	warnRecords := decodeJSONLines(t, readLogFile(t, warnFile))
	if len(warnRecords) != 1 {
		t.Fatalf("expected warn output to receive one record, got %d", len(warnRecords))
	}
	if warnRecords[0][0] != "@bunion:v1" || warnRecords[0][1] != "multi-json" || warnRecords[0][2] != "WARN" {
		t.Fatalf("unexpected warn record header: %#v", warnRecords[0][:3])
	}

	infoRecords := decodeJSONLines(t, readLogFile(t, infoFile))
	if len(infoRecords) != 2 {
		t.Fatalf("expected info output to receive two records, got %d", len(infoRecords))
	}
	if infoRecords[0][2] != "INFO" || infoRecords[1][2] != "WARN" {
		t.Fatalf("unexpected info output levels: %#v then %#v", infoRecords[0][2], infoRecords[1][2])
	}
}
