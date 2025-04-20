package melcloud

// Device represents a generic MELCloud device.
// Specific device types (ATA, ATW, ERV) will embed or reference this.
type Device struct {
	DeviceID           int    `json:"DeviceID"`
	BuildingID         int    `json:"BuildingID"`
	DeviceName         string `json:"DeviceName"`
	MacAddress         string `json:"MacAddress"`
	SerialNumber       string `json:"SerialNumber"`
	AccessLevel        int    `json:"AccessLevel"`
	DeviceType         int    `json:"DeviceType"`
	WifiSignalStrength int    `json:"WifiSignalStrength"`

	// Configuration fields often nested under "Device" in pymelcloud
	// These might be better handled by a separate capabilities/config struct
	TemperatureIncrement float64 `json:"TemperatureIncrement"`
	MinTempHeat          float64 `json:"MinTempHeat"`
	MaxTempHeat          float64 `json:"MaxTempHeat"`
	MinTempCoolDry       float64 `json:"MinTempCoolDry"`
	MaxTempCoolDry       float64 `json:"MaxTempCoolDry"`
	MinTempAutomatic     float64 `json:"MinTempAutomatic"`
	MaxTempAutomatic     float64 `json:"MaxTempAutomatic"`
	// Add other relevant conf fields...
}

// TODO: Potentially add methods to Device to fetch capabilities if needed.
