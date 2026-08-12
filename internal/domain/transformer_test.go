package domain

import (
	"errors"
	"testing"
)

func TestTransformerValidate(t *testing.T) {
	ok := Transformer{
		ID:                "TR-001",
		RatedPowerMVA:     40,
		HVVoltageKV:       230,
		LVVoltageKV:       13.8,
		FrequencyHz:       60,
		PhaseCount:        3,
		VectorGroup:       "YNd11",
		ImpedancePercent:  12,
		CoolingType:       CoolingONAF,
		CommissioningYear: 2018,
		Application:       ApplicationTransmission,
		NoLoadLossKW:      55,
		LoadLossKW:        380,
		TotalMassT:        68,
		LengthM:           5.4,
		WidthM:            3.3,
		HeightM:           4.1,
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid transformer, got: %v", err)
	}
}

func TestTransformerValidateErrors(t *testing.T) {
	base := Transformer{
		ID:                "TR-001",
		RatedPowerMVA:     40,
		HVVoltageKV:       230,
		LVVoltageKV:       13.8,
		FrequencyHz:       60,
		PhaseCount:        3,
		VectorGroup:       "YNd11",
		ImpedancePercent:  12,
		CoolingType:       CoolingONAF,
		CommissioningYear: 2018,
		Application:       ApplicationTransmission,
		NoLoadLossKW:      55,
		LoadLossKW:        380,
		TotalMassT:        68,
		LengthM:           5.4,
		WidthM:            3.3,
		HeightM:           4.1,
	}

	mut := func(f func(*Transformer)) Transformer {
		t := base
		f(&t)
		return t
	}

	cases := []struct {
		name string
		t    Transformer
		want error
	}{
		{"empty id", mut(func(t *Transformer) { t.ID = "" }), ErrEmptyID},
		{"malformed id", mut(func(t *Transformer) { t.ID = "TR-01" }), ErrInvalidID},
		{"zero power", mut(func(t *Transformer) { t.RatedPowerMVA = 0 }), ErrNonPositivePower},
		{"negative power", mut(func(t *Transformer) { t.RatedPowerMVA = -5 }), ErrNonPositivePower},
		{"negative voltage", mut(func(t *Transformer) { t.HVVoltageKV = -1 }), ErrVoltageOrder},
		{"lv above hv", mut(func(t *Transformer) { t.LVVoltageKV = 240 }), ErrVoltageOrder},
		{"bad frequency", mut(func(t *Transformer) { t.FrequencyHz = 55 }), ErrInvalidFrequency},
		{"bad phase count", mut(func(t *Transformer) { t.PhaseCount = 2 }), ErrInvalidPhaseCount},
		{"bad vector group", mut(func(t *Transformer) { t.VectorGroup = "XX0" }), ErrInvalidVectorGroup},
		{"zero impedance", mut(func(t *Transformer) { t.ImpedancePercent = 0 }), ErrImpedanceRange},
		{"huge impedance", mut(func(t *Transformer) { t.ImpedancePercent = 55 }), ErrImpedanceRange},
		{"bad cooling", mut(func(t *Transformer) { t.CoolingType = "ABC" }), ErrInvalidCoolingType},
		{"too old", mut(func(t *Transformer) { t.CommissioningYear = 1950 }), ErrCommissioningYear},
		{"negative no load loss", mut(func(t *Transformer) { t.NoLoadLossKW = -1 }), ErrNegativeLosses},
		{"zero mass", mut(func(t *Transformer) { t.TotalMassT = 0 }), ErrNonPositiveDimensions},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.t.Validate(); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}
