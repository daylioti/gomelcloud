# melcloud-go

[![Go Reference](https://pkg.go.dev/badge/github.com/daylioti/melcloud-go.svg)](https://pkg.go.dev/github.com/daylioti/melcloud-go)

A Go library for interacting with Mitsubishi Electric MELCloud devices (currently focused on Air-to-Air / ATA units).

This library aims to replicate the core functionality of the Python library `pymelcloud` for controlling MELCloud-enabled air conditioners and potentially other devices.

**Note:** This library is under development and interacts with an unofficial API. Use at your own risk. Breaking changes may occur.

## Features

*   **Authentication:** Login to MELCloud using email and password.
*   **Device Listing:** List all devices associated with the account.
*   **Get State (ATA):** Fetch the current detailed state of Air-to-Air (split system) air conditioners.
*   **Set State (ATA):** Control basic properties of ATA units:
    *   Power (On/Off)
    *   Target Temperature
    *   Operation Mode (Cool, Heat, Dry, Fan Only, Auto)
    *   Fan Speed (Auto, 1, 2, ...)
    *   Vane Positions (Vertical & Horizontal: Auto, 1-5, Swing, Split)

## Setup

1.  **Go Modules:** Ensure you have Go installed (version 1.18 or later recommended).
    Initialize your project if you haven't already:
    ```bash
    go mod init your_project_name
    ```
    Add this library as a dependency:
    ```bash
    go get github.com/daylioti/melcloud-go
    go mod tidy
    ```

2.  **Environment Variables:** The library requires your MELCloud credentials to be set as environment variables:
    ```bash
    export MELCLOUD_EMAIL="your_email@example.com"
    export MELCLOUD_PASSWORD="your_password"
    ```

## Usage Example

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/daylioti/melcloud-go" // Import from GitHub
)

func main() {
	// Ensure credentials are set (optional, Login() checks this too)
	if os.Getenv("MELCLOUD_EMAIL") == "" || os.Getenv("MELCLOUD_PASSWORD") == "" {
		log.Fatal("MELCLOUD_EMAIL and MELCLOUD_PASSWORD environment variables must be set")
	}

	// Login
	client, err := melcloud.Login()
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Println("Login successful!")

	// List Devices
	devices, err := client.ListDevices()
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}

	if len(devices) == 0 {
		fmt.Println("No devices found on this account.")
		return
	}

	fmt.Printf("Found %d devices:\n", len(devices))
	for i, device := range devices {
		fmt.Printf("  [%d] ID: %d, Name: %s, Type: %d\n", i, device.DeviceID, device.DeviceName, device.DeviceType)
	}

	// --- Interact with the first ATA device found ---
	var firstAtaDevice *melcloud.Device
	for i := range devices {
		if devices[i].DeviceType == 0 { // 0 = ATA
			firstAtaDevice = &devices[i]
			break
		}
	}

	if firstAtaDevice == nil {
		fmt.Println("No Air-to-Air (ATA) devices found.")
		return
	}

	fmt.Printf("\nInteracting with device: %s (ID: %d)\n", firstAtaDevice.DeviceName, firstAtaDevice.DeviceID)

	// Get Current State
	currentState, err := client.GetDeviceState(firstAtaDevice.DeviceID, firstAtaDevice.BuildingID)
	if err != nil {
		log.Fatalf("Failed to get device state: %v", err)
	}

	fmt.Printf("Current State: Power=%t, Mode=%s, Temp=%.1f, Fan=%s\n",
		currentState.Power,
		currentState.OperationModeString(),
		currentState.SetTemperature,
		currentState.FanSpeedString(),
	)

	// Set New State (Example: Turn on, set to Cool, 22C, Fan Auto)
	newState := *currentState // Start with a copy of the current state
	newState.ResetEffectiveFlags() // Clear flags before setting new properties

	newState.SetPower(true)
	err = newState.SetOperationMode(melcloud.ModeCool)
	if err != nil {
		log.Printf("Warning: Failed to set mode: %v", err)
	} else {
		newState.SetTargetTemperature(22.0)
		err = newState.SetFanSpeedMode(melcloud.FanAuto)
		if err != nil {
			log.Printf("Warning: Failed to set fan speed: %v", err)
		}
	}

	// Only send if we made valid changes
	if newState.EffectiveFlags > 0 {
		fmt.Println("Sending command to change state...")
		updatedState, err := client.SetDeviceState(newState)
		if err != nil {
			log.Fatalf("Failed to set device state: %v", err)
		}
		fmt.Printf("State after set: Power=%t, Mode=%s, Temp=%.1f, Fan=%s\n",
			updatedState.Power,
			updatedState.OperationModeString(),
			updatedState.SetTemperature,
			updatedState.FanSpeedString(),
		)
	} else {
		fmt.Println("No valid changes made to state, skipping SetDeviceState.")
	}
}

```

## Running Tests

Tests require your MELCloud credentials to be set as environment variables (`MELCLOUD_EMAIL`, `MELCLOUD_PASSWORD`).

**Warning:** `TestSetDeviceState` *will* modify the state of the first listed device on your account (it attempts to change the target temperature).

```bash
export MELCLOUD_EMAIL="your_email@example.com"
export MELCLOUD_PASSWORD="your_password"
go test -v .
```

## Limitations & TODOs

*   **ATA Focus:** Currently only tested and fully implemented for Air-to-Air (ATA) devices.
*   **Capabilities:** Does not yet parse or use device-specific capabilities (e.g., exact min/max temperatures, available fan speeds, temperature increment for rounding).
*   **Energy Reporting:** Not implemented.
*   **Other Device Types:** ATW and ERV devices are not supported.
*   **Error Handling:** API error details could be parsed more thoroughly.
*   **Rate Limiting:** Does not implement client-side rate limiting (be mindful of how often you call `GetDeviceState`).
*   **Async/Debounce:** Does not replicate `pymelcloud`'s async update loop or `set` debouncing. 

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
