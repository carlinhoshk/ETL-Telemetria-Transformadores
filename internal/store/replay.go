package store

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"etl-telemetria-transformadores/internal/ingestion"
	"etl-telemetria-transformadores/internal/telemetry"
)

// ReplayBronze reads a JSONL bronze dump and persists every record through the
// ingestion Store (raw provenance + normalized measurement). Lines are
// classified by the presence of the "payload" field (raw records carry it).
func ReplayBronze(ctx context.Context, sink ingestion.Store, r io.Reader) (rawN, measN int, err error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		line := sc.Bytes()
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(line, &probe); err != nil {
			return rawN, measN, fmt.Errorf("parse line: %w", err)
		}
		if _, isRaw := probe["payload"]; isRaw {
			var rec ingestion.RawRecord
			if err := json.Unmarshal(line, &rec); err != nil {
				return rawN, measN, fmt.Errorf("parse raw record: %w", err)
			}
			if err := sink.WriteRaw(ctx, rec); err != nil {
				return rawN, measN, fmt.Errorf("write raw %s: %w", rec.ID, err)
			}
			rawN++
			continue
		}
		var m telemetry.Measurement
		if err := json.Unmarshal(line, &m); err != nil {
			return rawN, measN, fmt.Errorf("parse measurement: %w", err)
		}
		if err := sink.WriteMeasurement(ctx, m); err != nil {
			return rawN, measN, fmt.Errorf("write measurement %s: %w", m.TransformerID, err)
		}
		measN++
	}
	return rawN, measN, sc.Err()
}
