package melcloud

import (
	"fmt"
	"strconv"
	"time"
)

// AtaDeviceState holds the detailed state of an Air-to-Air (ATA) device.
// This combines fields from the base device state and ATA specific ones.
type AtaDeviceState struct {
	// Base device fields (subset also available in the main GET response)
	DeviceID          int     `json:"DeviceID"`
	BuildingID        int     `json:"BuildingID"` // Note: Not always in Get response, use from Device struct
	MacAddress        string  `json:"MacAddress"`
	SerialNumber      string  `json:"SerialNumber"`
	DeviceType        int     `json:"DeviceType"` // 0 for ATA
	Power             bool    `json:"Power"`
	RoomTemperature   float64 `json:"RoomTemperature"`
	SetTemperature    float64 `json:"SetTemperature"`
	OperationMode     int     `json:"OperationMode"` // 1:Heat, 2:Dry, 3:Cool, 7:Fan, 8:Auto
	SetFanSpeed       int     `json:"SetFanSpeed"`   // 0:Auto, 1-N: Speeds
	VaneHorizontal    int     `json:"VaneHorizontal"`
	VaneVertical      int     `json:"VaneVertical"`
	ErrorCode         int     `json:"ErrorCode"`
	HasError          bool    `json:"HasError"`
	LastCommunication string  `json:"LastCommunication"` // ISO 8601 format "YYYY-MM-DDTHH:MM:SS.ffffff"
	EffectiveFlags    int     `json:"EffectiveFlags"`    // Crucial for setting state
	HasPendingCommand bool    `json:"HasPendingCommand"` // Crucial for setting state

	// Add other fields observed in API responses or pymelcloud as needed
	// e.g., OutdoorTemperature, NumberOfFanSpeeds, ActualFanSpeed etc.
}

// LastCommunicationTime parses the LastCommunication string into a time.Time object.
func (s *AtaDeviceState) LastCommunicationTime() (time.Time, error) {
	// MELCloud uses a specific format, sometimes with 6 or 7 fractional digits
	// We need to handle potential variations
	layout := "2006-01-02T15:04:05.000000"
	if len(s.LastCommunication) > len(layout) {
		// Adjust layout if more precision is present (e.g., .1234567)
		layout += "Z" // Assuming UTC if timezone not specified, adjust if needed
		return time.Parse(layout[:len(s.LastCommunication)], s.LastCommunication)
	} else if len(s.LastCommunication) < len(layout) {
		// Adjust layout if less precision is present
		return time.Parse(layout[:len(s.LastCommunication)], s.LastCommunication)
	}
	return time.Parse(layout, s.LastCommunication)
}

// Constants for ATA device properties
const (
	// EffectiveFlags indicate which properties are being set
	FlagPower          = 0x01
	FlagOperationMode  = 0x02
	FlagTargetTemp     = 0x04
	FlagFanSpeed       = 0x08
	FlagVaneVertical   = 0x10
	FlagVaneHorizontal = 0x100

	// Operation Modes (int)
	OpModeHeat     = 1
	OpModeDry      = 2
	OpModeCool     = 3
	OpModeFanOnly  = 7
	OpModeHeatCool = 8  // Auto
	OpModeUnknown  = -1 // Or some other indicator

	// Fan Speeds (int)
	FanSpeedAuto = 0

	// Vane Vertical Positions (int)
	VaneVertAuto    = 0
	VaneVert1       = 1
	VaneVert2       = 2
	VaneVert3       = 3
	VaneVert4       = 4
	VaneVert5       = 5
	VaneVertSwing   = 7
	VaneVertUnknown = -1

	// Vane Horizontal Positions (int)
	VaneHorizAuto    = 0
	VaneHoriz1       = 1
	VaneHoriz2       = 2
	VaneHoriz3       = 3
	VaneHoriz4       = 4
	VaneHoriz5       = 5
	VaneHorizSplit   = 8
	VaneHorizSwing   = 12
	VaneHorizUnknown = -1
)

// String constants for user interaction (maps to the int constants above)
const (
	ModeHeat     = "heat"
	ModeDry      = "dry"
	ModeCool     = "cool"
	ModeFanOnly  = "fan_only"
	ModeHeatCool = "heat_cool"
	ModeUnknown  = "unknown"

	FanAuto = "auto"

	VaneAuto  = "auto"
	VaneSwing = "swing"
	VaneSplit = "split" // Horizontal only
)

var opModeIntToString = map[int]string{
	OpModeHeat:     ModeHeat,
	OpModeDry:      ModeDry,
	OpModeCool:     ModeCool,
	OpModeFanOnly:  ModeFanOnly,
	OpModeHeatCool: ModeHeatCool,
}

var opModeStringToInt = map[string]int{
	ModeHeat:     OpModeHeat,
	ModeDry:      OpModeDry,
	ModeCool:     OpModeCool,
	ModeFanOnly:  OpModeFanOnly,
	ModeHeatCool: OpModeHeatCool,
}

// OperationModeString returns the string representation of the current operation mode.
func (s *AtaDeviceState) OperationModeString() string {
	if mode, ok := opModeIntToString[s.OperationMode]; ok {
		return mode
	}
	return ModeUnknown
}

// SetPower updates the Power state and sets the corresponding EffectiveFlag.
func (s *AtaDeviceState) SetPower(power bool) {
	s.Power = power
	s.EffectiveFlags |= FlagPower
}

// SetOperationMode updates the OperationMode from a string representation and sets the flag.
// Returns an error if the mode string is invalid.
func (s *AtaDeviceState) SetOperationMode(mode string) error {
	if modeInt, ok := opModeStringToInt[mode]; ok {
		s.OperationMode = modeInt
		s.EffectiveFlags |= FlagOperationMode
		return nil
	}
	return fmt.Errorf("invalid operation mode: %s", mode)
}

// SetTargetTemperature updates the SetTemperature and sets the corresponding EffectiveFlag.
// Note: Temperature rounding should be handled by the caller based on the Device's
// TemperatureIncrement field. For example:
//
//	if device.TemperatureIncrement > 0 {
//	    // Round to nearest increment
//	    temp = math.Round(temp/device.TemperatureIncrement) * device.TemperatureIncrement
//	}
//
func (s *AtaDeviceState) SetTargetTemperature(temp float64) {
	s.SetTemperature = temp
	s.EffectiveFlags |= FlagTargetTemp
}

// SetFanSpeedMode updates the SetFanSpeed field from a string representation ("auto", "1", "2", etc.)
// and sets the corresponding EffectiveFlag.
// Returns an error if the speed string is invalid.
func (s *AtaDeviceState) SetFanSpeedMode(speed string) error {
	if speed == FanAuto {
		s.SetFanSpeed = FanSpeedAuto // Assign to the field
		s.EffectiveFlags |= FlagFanSpeed
		return nil
	}
	// Try converting to integer
	speedInt, err := strconv.Atoi(speed)
	if err == nil && speedInt > 0 { // Assuming fan speeds are positive integers > 0
		// Note: Fan speed validation should be handled by the caller if needed.
		// For example, if you have access to the Device struct:
		//
		//   if device.NumberOfFanSpeeds > 0 && speedInt > device.NumberOfFanSpeeds {
		//       return fmt.Errorf("fan speed %d exceeds maximum of %d", speedInt, device.NumberOfFanSpeeds)
		//   }
		//
		s.SetFanSpeed = speedInt // Assign to the field
		s.EffectiveFlags |= FlagFanSpeed
		return nil
	}
	return fmt.Errorf("invalid fan speed: %s", speed)
}

// FanSpeedString returns the string representation ("auto", "1", "2", etc.) of the SetFanSpeed field.
func (s *AtaDeviceState) FanSpeedString() string {
	if s.SetFanSpeed == FanSpeedAuto { // Compare the field
		return FanAuto
	}
	return strconv.Itoa(s.SetFanSpeed) // Convert the field
}

// --- Vane Helpers ---

var vaneVertIntToString = map[int]string{
	VaneVertAuto:  VaneAuto,
	VaneVert1:     "1",
	VaneVert2:     "2",
	VaneVert3:     "3",
	VaneVert4:     "4",
	VaneVert5:     "5",
	VaneVertSwing: VaneSwing,
}

var vaneVertStringToInt = map[string]int{
	VaneAuto:  VaneVertAuto,
	"1":       VaneVert1,
	"2":       VaneVert2,
	"3":       VaneVert3,
	"4":       VaneVert4,
	"5":       VaneVert5,
	VaneSwing: VaneVertSwing,
}

var vaneHorizIntToString = map[int]string{
	VaneHorizAuto:  VaneAuto,
	VaneHoriz1:     "1",
	VaneHoriz2:     "2",
	VaneHoriz3:     "3",
	VaneHoriz4:     "4",
	VaneHoriz5:     "5",
	VaneHorizSplit: VaneSplit,
	VaneHorizSwing: VaneSwing,
}

var vaneHorizStringToInt = map[string]int{
	VaneAuto:  VaneHorizAuto,
	"1":       VaneHoriz1,
	"2":       VaneHoriz2,
	"3":       VaneHoriz3,
	"4":       VaneHoriz4,
	"5":       VaneHoriz5,
	VaneSplit: VaneHorizSplit,
	VaneSwing: VaneHorizSwing,
}

// VaneVerticalString returns the string representation ("auto", "1"-"5", "swing") of the VaneVertical field.
func (s *AtaDeviceState) VaneVerticalString() string {
	if pos, ok := vaneVertIntToString[s.VaneVertical]; ok {
		return pos
	}
	return "unknown" // Or constant VaneVertUnknown string
}

// SetVaneVertical updates the VaneVertical field from a string representation and sets the flag.
// Returns an error if the position string is invalid.
func (s *AtaDeviceState) SetVaneVertical(pos string) error {
	if posInt, ok := vaneVertStringToInt[pos]; ok {
		s.VaneVertical = posInt
		s.EffectiveFlags |= FlagVaneVertical
		return nil
	}
	return fmt.Errorf("invalid vertical vane position: %s", pos)
}

// VaneHorizontalString returns the string representation ("auto", "1"-"5", "split", "swing") of the VaneHorizontal field.
func (s *AtaDeviceState) VaneHorizontalString() string {
	if pos, ok := vaneHorizIntToString[s.VaneHorizontal]; ok {
		return pos
	}
	return "unknown" // Or constant VaneHorizUnknown string
}

// SetVaneHorizontal updates the VaneHorizontal field from a string representation and sets the flag.
// Returns an error if the position string is invalid.
func (s *AtaDeviceState) SetVaneHorizontal(pos string) error {
	if posInt, ok := vaneHorizStringToInt[pos]; ok {
		s.VaneHorizontal = posInt
		s.EffectiveFlags |= FlagVaneHorizontal
		return nil
	}
	return fmt.Errorf("invalid horizontal vane position: %s", pos)
}

// ResetEffectiveFlags clears the flags used for setting state.
// Useful after a successful SetDeviceState call or before setting new properties.
func (s *AtaDeviceState) ResetEffectiveFlags() {
	s.EffectiveFlags = 0
}
