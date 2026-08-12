// Command simulator models a fleet of transformers and emits telemetry that
// follows physics-plausible relationships. In Phase 2 output goes to stdout as
// JSON Lines; Phase 3 streams the same measurements over MQTT.
//
//	go run ./cmd/simulator -n 4 -interval 5 -seed 42 -intensity 1.0 -ticks 3
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/telemetry"
)

func main() {
	csvPath := flag.String("csv", "dbt/seeds/transformers.csv", "fleet CSV (design base)")
	n := flag.Int("n", 0, "max number of transformers to simulate (0 = all)")
	intervalSec := flag.Int("interval", 5, "simulation interval in seconds")
	seed := flag.Int64("seed", 42, "random seed")
	intensity := flag.Float64("intensity", 1.0, "load intensity multiplier (1.0 nominal)")
	ticks := flag.Int("ticks", 5, "number of ticks to emit")
	flag.Parse()

	if *n < 0 {
		fmt.Fprintln(os.Stderr, "error: -n must be >= 0")
		os.Exit(2)
	}
	if *intervalSec <= 0 {
		fmt.Fprintln(os.Stderr, "error: -interval must be > 0")
		os.Exit(2)
	}
	if *ticks <= 0 {
		fmt.Fprintln(os.Stderr, "error: -ticks must be > 0")
		os.Exit(2)
	}

	fleet, err := loadFleet(*csvPath, *n)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	sim, err := telemetry.New(telemetry.Config{
		Interval:      time.Duration(*intervalSec) * time.Second,
		Seed:          *seed,
		LoadIntensity: *intensity,
	}, fleet, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	for i := 0; i < *ticks; i++ {
		batch, err := sim.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		for _, m := range batch {
			if err := enc.Encode(m); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

func loadFleet(path string, limit int) ([]domain.Transformer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open fleet %s: %w", path, err)
	}
	defer f.Close()

	fleet, err := domain.LoadTransformerCSV(f)
	if err != nil {
		return nil, fmt.Errorf("parse fleet: %w", err)
	}
	if limit > 0 && limit < len(fleet) {
		fleet = fleet[:limit]
	}
	return fleet, nil
}
