package domain

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

// LoadTransformerCSV reads transformer design records from the dbt seed CSV.
func LoadTransformerCSV(r io.Reader) ([]Transformer, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	col := make(map[string]int, len(header))
	for i, name := range header {
		col[name] = i
	}

	required := []string{
		"transformer_id", "rated_power_mva", "hv_voltage_kv", "lv_voltage_kv",
		"frequency_hz", "phase_count", "vector_group", "impedance_percent",
		"cooling_type", "commissioning_year", "application",
		"no_load_loss_kw", "load_loss_kw", "total_mass_t", "length_m",
		"width_m", "height_m",
	}
	for _, name := range required {
		if _, ok := col[name]; !ok {
			return nil, fmt.Errorf("csv missing required column %q", name)
		}
	}

	var out []Transformer
	row := 1
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv row %d: %w", row, err)
		}
		row++

		tr := Transformer{
			ID:          rec[col["transformer_id"]],
			VectorGroup: VectorGroup(rec[col["vector_group"]]),
			CoolingType: CoolingType(rec[col["cooling_type"]]),
			Application: Application(rec[col["application"]]),
		}
		if tr.RatedPowerMVA, err = f(rec[col["rated_power_mva"]]); err != nil {
			return nil, fmt.Errorf("csv row %d rated_power_mva: %w", row, err)
		}
		if tr.HVVoltageKV, err = f(rec[col["hv_voltage_kv"]]); err != nil {
			return nil, fmt.Errorf("csv row %d hv_voltage_kv: %w", row, err)
		}
		if tr.LVVoltageKV, err = f(rec[col["lv_voltage_kv"]]); err != nil {
			return nil, fmt.Errorf("csv row %d lv_voltage_kv: %w", row, err)
		}
		if tr.FrequencyHz, err = i(rec[col["frequency_hz"]]); err != nil {
			return nil, fmt.Errorf("csv row %d frequency_hz: %w", row, err)
		}
		if tr.PhaseCount, err = i(rec[col["phase_count"]]); err != nil {
			return nil, fmt.Errorf("csv row %d phase_count: %w", row, err)
		}
		if tr.ImpedancePercent, err = f(rec[col["impedance_percent"]]); err != nil {
			return nil, fmt.Errorf("csv row %d impedance_percent: %w", row, err)
		}
		if tr.CommissioningYear, err = i(rec[col["commissioning_year"]]); err != nil {
			return nil, fmt.Errorf("csv row %d commissioning_year: %w", row, err)
		}
		if tr.NoLoadLossKW, err = f(rec[col["no_load_loss_kw"]]); err != nil {
			return nil, fmt.Errorf("csv row %d no_load_loss_kw: %w", row, err)
		}
		if tr.LoadLossKW, err = f(rec[col["load_loss_kw"]]); err != nil {
			return nil, fmt.Errorf("csv row %d load_loss_kw: %w", row, err)
		}
		if tr.TotalMassT, err = f(rec[col["total_mass_t"]]); err != nil {
			return nil, fmt.Errorf("csv row %d total_mass_t: %w", row, err)
		}
		if tr.LengthM, err = f(rec[col["length_m"]]); err != nil {
			return nil, fmt.Errorf("csv row %d length_m: %w", row, err)
		}
		if tr.WidthM, err = f(rec[col["width_m"]]); err != nil {
			return nil, fmt.Errorf("csv row %d width_m: %w", row, err)
		}
		if tr.HeightM, err = f(rec[col["height_m"]]); err != nil {
			return nil, fmt.Errorf("csv row %d height_m: %w", row, err)
		}

		if err := tr.Validate(); err != nil {
			return nil, fmt.Errorf("csv row %d invalid transformer: %w", row, err)
		}
		out = append(out, tr)
	}
	return out, nil
}

func f(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func i(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return v, nil
}
