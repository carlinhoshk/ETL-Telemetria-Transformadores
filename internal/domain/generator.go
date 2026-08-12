package domain

import (
	"fmt"
	"math"
	"math/rand"
)

// Generator produces a synthetic fleet of transformer design records. It is
// deterministic for a given seed and biases values toward physical
// plausibility (power scale, voltage step pairings, impedance, cooling,
// losses, mass and dimensions). No values are purely random.
type Generator struct {
	rnd *rand.Rand
}

// NewGenerator returns a Generator seeded for reproducibility.
func NewGenerator(seed int64) *Generator {
	return &Generator{rnd: rand.New(rand.NewSource(seed))}
}

// designEnv describes a plausible envelope for a given application.
type designEnv struct {
	minPowerMVA float64
	maxPowerMVA float64
	hvKVs       []float64
	lvKVs       []float64
	vectorGroup []VectorGroup
}

var envelopeByApplication = map[Application]designEnv{
	ApplicationGeneration: {
		minPowerMVA: 30, maxPowerMVA: 400,
		hvKVs:       []float64{115, 138, 230, 345, 500},
		lvKVs:       []float64{11, 13.8, 20, 34.5, 69},
		vectorGroup: []VectorGroup{"YNd11", "YNd5", "Yd11", "Ynd5"},
	},
	ApplicationTransmission: {
		minPowerMVA: 50, maxPowerMVA: 400,
		hvKVs:       []float64{230, 345, 500},
		lvKVs:       []float64{69, 115, 138},
		vectorGroup: []VectorGroup{"YNd11", "YNd1", "Yd11"},
	},
	ApplicationDistribution: {
		minPowerMVA: 5, maxPowerMVA: 60,
		hvKVs:       []float64{69, 138},
		lvKVs:       []float64{13.8, 34.5},
		vectorGroup: []VectorGroup{"Dyn1", "Yy0", "Dd0", "Zyn11"},
	},
	ApplicationIndustrial: {
		minPowerMVA: 10, maxPowerMVA: 80,
		hvKVs:       []float64{69, 138, 230},
		lvKVs:       []float64{13.8, 34.5, 69},
		vectorGroup: []VectorGroup{"Dyn1", "YNd11", "Dd0"},
	},
	ApplicationRenewable: {
		minPowerMVA: 20, maxPowerMVA: 120,
		hvKVs:       []float64{34.5, 69, 138},
		lvKVs:       []float64{13.8, 34.5},
		vectorGroup: []VectorGroup{"Dyn1", "YNd11", "Dd0"},
	},
}

// Generate creates n transformers. Every record is validated; an error reports
// the violating index and record.
func (g *Generator) Generate(n int) ([]Transformer, error) {
	fleet := make([]Transformer, 0, n)
	for i := 0; i < n; i++ {
		t := g.one(i + 1)
		if err := t.Validate(); err != nil {
			return nil, fmt.Errorf("generated invalid transformer #%d (%s): %w", i+1, t.ID, err)
		}
		fleet = append(fleet, t)
	}
	return fleet, nil
}

func (g *Generator) one(seq int) Transformer {
	application := ValidApplications[g.i(len(ValidApplications))]
	env := envelopeByApplication[application]

	mva := g.logUniform(env.minPowerMVA, env.maxPowerMVA)
	hv, lv := g.voltagePair(supportedHVKVs(mva, env.hvKVs), env.lvKVs)

	return Transformer{
		ID:                fmt.Sprintf("TR-%03d", seq),
		RatedPowerMVA:     round1(mva),
		HVVoltageKV:       hv,
		LVVoltageKV:       lv,
		FrequencyHz:       g.frequency(),
		PhaseCount:        g.phaseCount(),
		VectorGroup:       env.vectorGroup[g.i(len(env.vectorGroup))],
		ImpedancePercent:  round1(g.impedance(mva)),
		CoolingType:       g.cooling(mva),
		CommissioningYear: g.year(),
		Application:       application,
		NoLoadLossKW:      round1(mva * 1000 * g.uniform(0.0008, 0.0016)),
		LoadLossKW:        round1(mva * 1000 * g.uniform(0.006, 0.012)),
		TotalMassT:        round1(g.mass(mva)),
		LengthM:           round2(g.length(mva)),
		WidthM:            round2(g.width(mva)),
		HeightM:           round2(g.height(mva)),
	}
}

// supportedHVKVs restricts very high voltage classes to units large enough
// to justify them, keeping the synthetic fleet engineering-plausible.
func supportedHVKVs(mva float64, kvs []float64) []float64 {
	valid := make([]float64, 0, len(kvs))
	for _, kv := range kvs {
		if kv >= 345 && mva < 80 {
			continue // 345/500 kV class needs ~100 MVA+
		}
		if kv >= 230 && mva < 40 {
			continue
		}
		valid = append(valid, kv)
	}
	if len(valid) == 0 {
		return kvs
	}
	return valid
}

func (g *Generator) voltagePair(hvs, lvs []float64) (float64, float64) {
	hv := hvs[g.i(len(hvs))]
	// Only pick LV below the chosen HV so the group substitution rule holds.
	valid := make([]float64, 0, len(lvs))
	for _, lv := range lvs {
		if lv < hv {
			valid = append(valid, lv)
		}
	}
	if len(valid) == 0 {
		return hv, lvs[g.i(len(lvs))] // fallback, validation will catch it
	}
	return hv, valid[g.i(len(valid))]
}

// impedance grows with rated power, reflecting the classic design trade-off,
// with a small jitter.
func (g *Generator) impedance(mva float64) float64 {
	base := 5.0 + 5.0*math.Log10(mva)
	if base > 25 {
		base = 25
	}
	return base * g.uniform(0.9, 1.1)
}

func (g *Generator) cooling(mva float64) CoolingType {
	switch {
	case mva < 15:
		return CoolingONAN
	case mva < 60:
		return CoolingONAF
	case mva < 150:
		return []CoolingType{CoolingONAF, CoolingOFAF}[g.i(2)]
	default:
		return []CoolingType{CoolingOFAF, CoolingODAF, CoolingOFWF}[g.i(3)]
	}
}

func (g *Generator) frequency() int {
	if g.rnd.Float64() < 0.9 {
		return 60
	}
	return 50
}

func (g *Generator) phaseCount() int {
	if g.rnd.Float64() < 0.99 {
		return 3
	}
	return 1
}

func (g *Generator) year() int {
	// Skewed toward recent units using sqrt of a uniform draw.
	u := math.Sqrt(g.rnd.Float64()) // 0..1, denser near 1
	return 1990 + int(u*36)         // 1990..2025
}

func (g *Generator) mass(mva float64) float64 {
	return 22 + 1.35*math.Pow(mva, 0.9)*g.uniform(0.85, 1.15)
}

func (g *Generator) length(mva float64) float64 {
	return 1.8 + 0.5*math.Sqrt(mva)*g.uniform(0.9, 1.1)
}

func (g *Generator) width(mva float64) float64 {
	return 1.4 + 0.3*math.Sqrt(mva)*g.uniform(0.9, 1.1)
}

func (g *Generator) height(mva float64) float64 {
	return 2.6 + 0.25*math.Sqrt(mva)*g.uniform(0.9, 1.1)
}

func (g *Generator) logUniform(min, max float64) float64 {
	return math.Exp(math.Log(min) + g.rnd.Float64()*(math.Log(max)-math.Log(min)))
}

func (g *Generator) uniform(lo, hi float64) float64 {
	return lo + g.rnd.Float64()*(hi-lo)
}

func (g *Generator) i(n int) int {
	return g.rnd.Intn(n)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
