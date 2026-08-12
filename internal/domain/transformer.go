// Package domain holds the core transformer concepts: the design/project
// record that forms the historical project base used by the similarity
// mechanism, and the telemetry envelope shared across the platform.
package domain

import (
	"errors"
	"fmt"
	"regexp"
)

// Application describes where a transformer is used. It shapes the design
// envelope of plausible synthetic projects.
type Application string

const (
	ApplicationGeneration   Application = "generation"
	ApplicationTransmission Application = "transmission"
	ApplicationDistribution Application = "distribution"
	ApplicationIndustrial   Application = "industrial"
	ApplicationRenewable    Application = "renewable"
)

// ValidApplications lists the supported applications.
var ValidApplications = []Application{
	ApplicationGeneration,
	ApplicationTransmission,
	ApplicationDistribution,
	ApplicationIndustrial,
	ApplicationRenewable,
}

// CoolingType follows the standard transformer cooling notation
// (ONAN, ONAF, OFAF, OFWF, ODAF).
type CoolingType string

const (
	CoolingONAN CoolingType = "ONAN"
	CoolingONAF CoolingType = "ONAF"
	CoolingOFAF CoolingType = "OFAF"
	CoolingOFWF CoolingType = "OFWF"
	CoolingODAF CoolingType = "ODAF"
)

// ValidCoolingTypes lists the supported cooling types.
var ValidCoolingTypes = []CoolingType{
	CoolingONAN,
	CoolingONAF,
	CoolingOFAF,
	CoolingOFWF,
	CoolingODAF,
}

// VectorGroup is the winding connection vector group (e.g. "YNd11", "Dyn1").
type VectorGroup string

// ValidVectorGroups lists commonly used vector groups.
var ValidVectorGroups = []VectorGroup{
	"YNd11", "YNd1", "Ynd5", "Yy0", "Dyn1", "Dd0", "YNd5", "Yd11", "Zyn11",
}

// Transformer is the design/project record of a power transformer. It is the
// unit of the historical project base and the input for project similarity.
type Transformer struct {
	ID                string      `json:"transformer_id"`
	RatedPowerMVA     float64     `json:"rated_power_mva"`
	HVVoltageKV       float64     `json:"hv_voltage_kv"`
	LVVoltageKV       float64     `json:"lv_voltage_kv"`
	FrequencyHz       int         `json:"frequency_hz"`
	PhaseCount        int         `json:"phase_count"`
	VectorGroup       VectorGroup `json:"vector_group"`
	ImpedancePercent  float64     `json:"impedance_percent"`
	CoolingType       CoolingType `json:"cooling_type"`
	CommissioningYear int         `json:"commissioning_year"`
	Application       Application `json:"application"`
	// Design extras, useful for a richer similarity and engineering context.
	NoLoadLossKW float64 `json:"no_load_loss_kw"`
	LoadLossKW   float64 `json:"load_loss_kw"`
	TotalMassT   float64 `json:"total_mass_t"`
	LengthM      float64 `json:"length_m"`
	WidthM       float64 `json:"width_m"`
	HeightM      float64 `json:"height_m"`
}

var (
	// ErrEmptyID indicates a missing transformer ID.
	ErrEmptyID = errors.New("transformer_id is required")
	// ErrInvalidID indicates an ID not matching the TR-XXX pattern.
	ErrInvalidID = errors.New("transformer_id must match TR-XXX")
	// ErrNonPositivePower indicates a non-positive rated power.
	ErrNonPositivePower = errors.New("rated_power_mva must be > 0")
	// ErrVoltageOrder indicates hv_voltage_kv is not greater than lv_voltage_kv.
	ErrVoltageOrder = errors.New("hv_voltage_kv must be > lv_voltage_kv")
	// ErrInvalidFrequency indicates a frequency other than 50 or 60 Hz.
	ErrInvalidFrequency = errors.New("frequency_hz must be 50 or 60")
	// ErrInvalidPhaseCount indicates a phase count other than 1 or 3.
	ErrInvalidPhaseCount = errors.New("phase_count must be 1 or 3")
	// ErrInvalidVectorGroup indicates an unknown vector group.
	ErrInvalidVectorGroup = errors.New("unknown vector_group")
	// ErrImpedanceRange indicates an out-of-range impedance.
	ErrImpedanceRange = errors.New("impedance_percent must be within (0, 40]")
	// ErrInvalidCoolingType indicates an unknown cooling type.
	ErrInvalidCoolingType = errors.New("unknown cooling_type")
	// ErrCommissioningYear indicates an impossible commissioning year.
	ErrCommissioningYear = errors.New("commissioning_year must be within [1960, current]")
	// ErrNegativeLosses indicates negative loss values.
	ErrNegativeLosses = errors.New("losses must be >= 0")
	// ErrNonPositiveDimensions indicates non-positive physical dimensions.
	ErrNonPositiveDimensions = errors.New("dimensions must be > 0")
)

var idPattern = regexp.MustCompile(`^TR-\d{3}$`)

// Validate checks the consistency of a transformer design record.
func (t *Transformer) Validate() error {
	if t.ID == "" {
		return ErrEmptyID
	}
	if !idPattern.MatchString(t.ID) {
		return ErrInvalidID
	}
	if t.RatedPowerMVA <= 0 {
		return ErrNonPositivePower
	}
	if t.HVVoltageKV <= 0 || t.LVVoltageKV <= 0 || t.HVVoltageKV <= t.LVVoltageKV {
		return ErrVoltageOrder
	}
	if t.FrequencyHz != 50 && t.FrequencyHz != 60 {
		return ErrInvalidFrequency
	}
	if t.PhaseCount != 1 && t.PhaseCount != 3 {
		return ErrInvalidPhaseCount
	}
	if !containsVectorGroup(t.VectorGroup) {
		return ErrInvalidVectorGroup
	}
	if t.ImpedancePercent <= 0 || t.ImpedancePercent > 40 {
		return ErrImpedanceRange
	}
	if !containsCoolingType(t.CoolingType) {
		return ErrInvalidCoolingType
	}
	if t.CommissioningYear < 1960 {
		return fmt.Errorf("%w: %d", ErrCommissioningYear, t.CommissioningYear)
	}
	if t.NoLoadLossKW < 0 || t.LoadLossKW < 0 {
		return ErrNegativeLosses
	}
	if t.TotalMassT <= 0 || t.LengthM <= 0 || t.WidthM <= 0 || t.HeightM <= 0 {
		return ErrNonPositiveDimensions
	}
	return nil
}

func containsVectorGroup(vg VectorGroup) bool {
	for _, v := range ValidVectorGroups {
		if v == vg {
			return true
		}
	}
	return false
}

func containsCoolingType(ct CoolingType) bool {
	for _, c := range ValidCoolingTypes {
		if c == ct {
			return true
		}
	}
	return false
}
