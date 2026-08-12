package telemetry

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"etl-telemetria-transformadores/internal/domain"
)

// Config tunes the simulation run.
type Config struct {
	// Interval between ticks.
	Interval time.Duration
	// Seed makes the whole run reproducible.
	Seed int64
	// LoadIntensity scales the base load profile (1.0 = nominal). Values above
	// 1.0 push units into overload states more often.
	LoadIntensity float64
}

// unitState holds the live thermal state of one transformer.
type unitState struct {
	oilTempC     float64
	windingTempC float64
	ambientC     float64
}

// Simulator models a fleet of transformers over time. It is deterministic
// for a given seed and start time: every draw (profile offset, noise) comes
// from a seeded PRNG, never from pure randomness.
type Simulator struct {
	cfg   Config
	rnd   *rand.Rand
	fleet []domain.Transformer
	units []unitState
	param map[string]unitParams
	now   time.Time
}

// New builds a Simulator for the fleet, warming thermal state from the start
// ambient and preparing deterministic per-unit parameters.
func New(cfg Config, fleet []domain.Transformer, start time.Time) (*Simulator, error) {
	if len(fleet) == 0 {
		return nil, fmt.Errorf("telemetry: empty fleet")
	}
	if cfg.Interval <= 0 {
		return nil, fmt.Errorf("telemetry: interval must be > 0")
	}
	intensity := cfg.LoadIntensity
	if intensity <= 0 {
		intensity = 1.0
	}

	s := &Simulator{
		cfg:   Config{Interval: cfg.Interval, Seed: cfg.Seed, LoadIntensity: intensity},
		rnd:   rand.New(rand.NewSource(cfg.Seed)),
		fleet: fleet,
		units: make([]unitState, len(fleet)),
		param: make(map[string]unitParams, len(fleet)),
		now:   start.UTC(),
	}

	ambient := ambientAt(s.now, s.rnd)
	for i, tr := range fleet {
		s.units[i] = unitState{
			ambientC:     ambient,
			oilTempC:     ambient,
			windingTempC: ambient,
		}
		s.param[tr.ID] = unitParams{
			maxOilRiseC:    oilRiseFor(tr.CoolingType) + s.jitter(2.0),
			windingGradC:   windingGradFor(tr.CoolingType) + s.jitter(1.5),
			lossRatio:      lossRatioOf(tr),
			ratedVoltageKV: tr.HVVoltageKV,
			loadOffsetH:    s.rnd.Float64() * 8, // ±4 h around the day
		}
	}
	return s, nil
}

func (s *Simulator) jitter(scale float64) float64 {
	return (s.rnd.Float64() - 0.5) * 2 * scale
}

// lossRatioOf approximates R = load_loss / no_load_loss from the design record.
func lossRatioOf(tr domain.Transformer) float64 {
	if tr.NoLoadLossKW <= 0 {
		return 8
	}
	return tr.LoadLossKW / tr.NoLoadLossKW
}

// ambientAt computes the daily ambient temperature at instant t.
func ambientAt(t time.Time, rnd *rand.Rand) float64 {
	hour := float64(t.Hour()) + float64(t.Minute())/60
	return ambientBaseC + ambientAmpC*math.Sin(2*math.Pi*(hour-14)/24)
}

// Next advances the simulation by one interval and returns a measurement per
// transformer, ordered like the fleet.
func (s *Simulator) Next() ([]Measurement, error) {
	s.now = s.now.Add(s.cfg.Interval)
	dt := s.cfg.Interval.Seconds()
	out := make([]Measurement, 0, len(s.fleet))

	for i, tr := range s.fleet {
		unit := &s.units[i]
		p := s.param[tr.ID]

		load := s.targetLoad(p.loadOffsetH, unit.ambientC)
		ambient := ambientAt(s.now, s.rnd)

		oilTarget := topOilSteady(ambient, load/100, p.lossRatio, p.maxOilRiseC)
		unit.oilTempC = thermalStep(unit.oilTempC, oilTarget, oilTimeConstant, dt)

		windingTarget := windingSteady(unit.oilTempC, load/100, p.windingGradC)
		unit.windingTempC = thermalStep(unit.windingTempC, windingTarget, windingTimeConstant, dt)
		unit.ambientC = ambient

		oilLevel := clamp(96+(unit.oilTempC-ambient)*0.03+s.jitter(0.4), 85, 99.8)
		current := deriveCurrent(load, tr.RatedPowerMVA, p.ratedVoltageKV)
		voltage := deriveVoltage(p.ratedVoltageKV, load)

		out = append(out, Measurement{
			SchemaVersion:   SchemaVersion,
			TransformerID:   tr.ID,
			Timestamp:       timestamp(s.now),
			LoadPercent:     round1(load),
			AmbientTempC:    round1(ambient),
			OilTempC:        round1(unit.oilTempC),
			WindingTempC:    round1(unit.windingTempC),
			OilLevelPercent: round1(oilLevel),
			CurrentA:        round1(current),
			VoltageKV:       round1(voltage),
			State:           classify(unit.windingTempC, unit.oilTempC, load),
		})
	}
	return out, nil
}

// targetLoad returns the load percent for a unit at the current time, bounded
// to plausible operating limits. High intensity pushes units into overload.
func (s *Simulator) targetLoad(offsetH, ambientC float64) float64 {
	hour := float64(s.now.Hour()) + float64(s.now.Minute())/60
	profile := baseLoad(hour, offsetH)

	// Slight weather coupling: warmer ambient increases cooling demand slightly.
	weather := 1 + (ambientC-ambientBaseC)*0.01

	noise := 1 + (s.rnd.Float64()-0.5)*0.06 // ±3 %
	return clamp(s.cfg.LoadIntensity*profile*weather*noise*100, 5, 160)
}
