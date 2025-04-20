package melcloud

import (
	"os"
	"testing"
)

// TestLogin requires MELCLOUD_EMAIL and MELCLOUD_PASSWORD environment variables to be set.
func TestLogin(t *testing.T) {
	if os.Getenv("MELCLOUD_EMAIL") == "" || os.Getenv("MELCLOUD_PASSWORD") == "" {
		t.Skip("Skipping Login test: MELCLOUD_EMAIL and/or MELCLOUD_PASSWORD not set")
	}

	client, err := Login()
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if client == nil {
		t.Fatal("Login returned nil client without error")
	}
	if client.token == "" {
		t.Error("Login returned client with empty token")
	}
	t.Logf("Login successful, token starts with: %s...", client.token[:min(10, len(client.token))])
}

// TestListDevices requires MELCLOUD_EMAIL and MELCLOUD_PASSWORD environment variables to be set.
func TestListDevices(t *testing.T) {
	if os.Getenv("MELCLOUD_EMAIL") == "" || os.Getenv("MELCLOUD_PASSWORD") == "" {
		t.Skip("Skipping ListDevices test: MELCLOUD_EMAIL and/or MELCLOUD_PASSWORD not set")
	}

	client, err := Login()
	if err != nil {
		t.Fatalf("Login failed during ListDevices test setup: %v", err)
	}

	devices, err := client.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}

	if devices == nil {
		t.Fatal("ListDevices returned nil slice without error")
	}

	// Just check that we got at least one device
	if len(devices) == 0 {
		t.Error("Expected at least one device, but got none")
	}

	t.Logf("Successfully listed %d devices:", len(devices))
	for _, device := range devices {
		t.Logf("  - ID: %d, Name: %s, BuildingID: %d", device.DeviceID, device.DeviceName, device.BuildingID)
	}
}

// TestGetDeviceState requires MELCLOUD_EMAIL and MELCLOUD_PASSWORD environment variables to be set.
func TestGetDeviceState(t *testing.T) {
	if os.Getenv("MELCLOUD_EMAIL") == "" || os.Getenv("MELCLOUD_PASSWORD") == "" {
		t.Skip("Skipping GetDeviceState test: MELCLOUD_EMAIL and/or MELCLOUD_PASSWORD not set")
	}

	client, err := Login()
	if err != nil {
		t.Fatalf("Login failed during GetDeviceState test setup: %v", err)
	}

	devices, err := client.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices failed during GetDeviceState test setup: %v", err)
	}

	if len(devices) == 0 {
		t.Fatal("No devices found to test GetDeviceState")
	}

	// Test with the first device found
	testDevice := devices[0]
	t.Logf("Attempting to get state for device ID %d (Name: %s, BuildingID: %d)", testDevice.DeviceID, testDevice.DeviceName, testDevice.BuildingID)

	state, err := client.GetDeviceState(testDevice.DeviceID, testDevice.BuildingID)
	if err != nil {
		t.Fatalf("GetDeviceState failed: %v", err)
	}

	if state == nil {
		t.Fatal("GetDeviceState returned nil state without error")
	}

	t.Logf("Successfully retrieved state for device %d:", testDevice.DeviceID)
	t.Logf("  Power: %t", state.Power)
	t.Logf("  Room Temp: %.1f", state.RoomTemperature)
	t.Logf("  Set Temp: %.1f", state.SetTemperature)
	t.Logf("  Op Mode: %d", state.OperationMode)
	t.Logf("  Fan Speed: %d", state.SetFanSpeed)
	t.Logf("  Vane H: %d, Vane V: %d", state.VaneHorizontal, state.VaneVertical)
	t.Logf("  EffectiveFlags: %d", state.EffectiveFlags)
	t.Logf("  HasPendingCommand: %t", state.HasPendingCommand)
	t.Logf("  ErrorCode: %d", state.ErrorCode)
	t.Logf("  HasError: %t", state.HasError)
	lastComm, err := state.LastCommunicationTime()
	if err != nil {
		t.Logf("  Last Comm: %s (parse error: %v)", state.LastCommunication, err)
	} else {
		t.Logf("  Last Comm: %s", lastComm)
	}
}

// TestSetDeviceState requires MELCLOUD_EMAIL and MELCLOUD_PASSWORD environment variables to be set.
// WARNING: This test modifies the state of your first listed device (sets temperature).
func TestSetDeviceState(t *testing.T) {
	if os.Getenv("MELCLOUD_EMAIL") == "" || os.Getenv("MELCLOUD_PASSWORD") == "" {
		t.Skip("Skipping SetDeviceState test: MELCLOUD_EMAIL and/or MELCLOUD_PASSWORD not set")
	}

	client, err := Login()
	if err != nil {
		t.Fatalf("Login failed during SetDeviceState test setup: %v", err)
	}

	devices, err := client.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices failed during SetDeviceState test setup: %v", err)
	}

	if len(devices) == 0 {
		t.Fatal("No devices found to test SetDeviceState")
	}

	// Use the first device found
	testDevice := devices[0]
	t.Logf("Using device ID %d (Name: %s) for SetDeviceState test", testDevice.DeviceID, testDevice.DeviceName)

	// 1. Get current state
	initialState, err := client.GetDeviceState(testDevice.DeviceID, testDevice.BuildingID)
	if err != nil {
		t.Fatalf("GetDeviceState failed before setting: %v", err)
	}
	if initialState == nil {
		t.Fatal("GetDeviceState returned nil initial state")
	}
	t.Logf("Initial state - Set Temp: %.1f, Power: %t, Flags: %d", initialState.SetTemperature, initialState.Power, initialState.EffectiveFlags)

	// 2. Modify the state (create a copy to avoid modifying the initial read)
	newState := *initialState      // Create a copy
	newState.ResetEffectiveFlags() // Reset flags before setting new properties

	// Example: Change temperature (adjust value as needed for a safe test)
	// Let's try setting it 0.5 degrees higher/lower than current, or to a fixed value like 23.0
	newTemp := 23.0
	if initialState.SetTemperature == newTemp {
		newTemp = 23.5 // If already 23.0, change to 23.5
	}
	t.Logf("Attempting to set temperature to: %.1f", newTemp)
	newState.SetTargetTemperature(newTemp)
	// You could chain other changes here, e.g.:
	// newState.SetPower(true)
	// err = newState.SetOperationMode(ModeCool)
	// if err != nil { t.Fatalf("Failed to set operation mode: %v", err) }

	t.Logf("State to send - Set Temp: %.1f, Power: %t, Flags: %d", newState.SetTemperature, newState.Power, newState.EffectiveFlags)

	// 3. Send the updated state
	returnedState, err := client.SetDeviceState(newState)
	if err != nil {
		t.Fatalf("SetDeviceState failed: %v", err)
	}
	if returnedState == nil {
		t.Fatal("SetDeviceState returned nil state without error")
	}

	// 4. Log the returned state (MELCloud might take time to fully update)
	t.Logf("State returned by API after set - Set Temp: %.1f, Power: %t, Flags: %d", returnedState.SetTemperature, returnedState.Power, returnedState.EffectiveFlags)

	// Optional: Add a small delay and get state again to verify (but MELCloud can be slow)
	// time.Sleep(5 * time.Second)
	// finalState, err := client.GetDeviceState(testDevice.DeviceID, testDevice.BuildingID)
	// ... check finalState ...

	// Basic check: Ensure the returned temperature matches what we tried to set
	if returnedState.SetTemperature != newTemp {
		t.Errorf("Returned state temperature (%.1f) does not match set temperature (%.1f)", returnedState.SetTemperature, newTemp)
	}
}

// Helper function for logging token prefix
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
