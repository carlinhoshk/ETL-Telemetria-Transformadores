// Command etl provides batch data tooling for the platform. In Phase 1 it
// seeds the historical transformer project base from the synthetic generator.
//
//	etl generate -n 20 -seed 42 -out data/transformers.json
//	etl generate -n 20 -seed 42 -out dbt/seeds/transformers.csv
package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"etl-telemetria-transformadores/internal/domain"
)

func main() {
	gen := flag.NewFlagSet("generate", flag.ExitOnError)
	n := gen.Int("n", 20, "number of transformers to generate")
	seed := gen.Int64("seed", 42, "random seed for reproducibility")
	out := gen.String("out", "data/transformers.json", "output path (.json or .csv)")

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: etl <generate> [flags]")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "generate":
		if err := gen.Parse(os.Args[2:]); err != nil {
			os.Exit(2)
		}
		if err := runGenerate(*n, *seed, *out); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

func runGenerate(n int, seed int64, out string) error {
	if n <= 0 {
		return errors.New("-n must be > 0")
	}
	fleet, err := domain.NewGenerator(seed).Generate(n)
	if err != nil {
		return err
	}

	if err := writeFleet(fleet, out); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}

	min, max := fleet[0].RatedPowerMVA, fleet[0].RatedPowerMVA
	for _, tr := range fleet {
		if tr.RatedPowerMVA < min {
			min = tr.RatedPowerMVA
		}
		if tr.RatedPowerMVA > max {
			max = tr.RatedPowerMVA
		}
	}
	fmt.Printf("generated %d transformers (seed=%d) -> %s\n", n, seed, out)
	fmt.Printf("power range: %.1f..%.1f MVA\n", min, max)
	return nil
}

func writeFleet(fleet []domain.Transformer, out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	switch filepath.Ext(out) {
	case ".csv":
		return writeCSV(fleet, out)
	default:
		return writeJSON(fleet, out)
	}
}

func writeJSON(fleet []domain.Transformer, out string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(fleet)
}

func writeCSV(fleet []domain.Transformer, out string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"transformer_id", "rated_power_mva", "hv_voltage_kv", "lv_voltage_kv",
		"frequency_hz", "phase_count", "vector_group", "impedance_percent",
		"cooling_type", "commissioning_year", "application",
		"no_load_loss_kw", "load_loss_kw", "total_mass_t", "length_m",
		"width_m", "height_m",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, tr := range fleet {
		row := []string{
			tr.ID,
			fmt.Sprintf("%.1f", tr.RatedPowerMVA),
			fmt.Sprintf("%.1f", tr.HVVoltageKV),
			fmt.Sprintf("%.1f", tr.LVVoltageKV),
			fmt.Sprintf("%d", tr.FrequencyHz),
			fmt.Sprintf("%d", tr.PhaseCount),
			string(tr.VectorGroup),
			fmt.Sprintf("%.1f", tr.ImpedancePercent),
			string(tr.CoolingType),
			fmt.Sprintf("%d", tr.CommissioningYear),
			string(tr.Application),
			fmt.Sprintf("%.1f", tr.NoLoadLossKW),
			fmt.Sprintf("%.1f", tr.LoadLossKW),
			fmt.Sprintf("%.1f", tr.TotalMassT),
			fmt.Sprintf("%.2f", tr.LengthM),
			fmt.Sprintf("%.2f", tr.WidthM),
			fmt.Sprintf("%.2f", tr.HeightM),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
