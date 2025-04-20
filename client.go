package melcloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	baseURL    = "https://app.melcloud.com/Mitsubishi.Wifi.Client"
	appVersion = "1.19.1.1"
)

// LoginResponse represents the structure of the login API response.
type LoginResponse struct {
	ErrorId      interface{} `json:"ErrorId"` // Using interface{} as it can be null
	ErrorCode    interface{} `json:"ErrorCode"`
	LoginData    LoginData   `json:"LoginData"`
	LoginMinutes int         `json:"LoginMinutes"`
	NextURLL     interface{} `json:"NextURLL"` // Assuming typo, might be NextURL
}

// LoginData contains the authentication context key.
type LoginData struct {
	ContextKey string `json:"ContextKey"`
	// Add other fields if needed
}

// Structure holds Areas and Floors which contain Devices
type Structure struct {
	Devices []Device `json:"Devices"`
	Areas   []Area   `json:"Areas"`
	Floors  []Floor  `json:"Floors"`
}

// Area contains Devices
type Area struct {
	Devices []Device `json:"Devices"`
}

// Floor contains Devices and Areas
type Floor struct {
	Devices []Device `json:"Devices"`
	Areas   []Area   `json:"Areas"`
}

// Building represents a building containing devices.
type Building struct {
	Structure Structure `json:"Structure"`
	// Add other Building fields if needed
}

// Client holds the API client state, including the auth token.
type Client struct {
	token      string
	httpClient *http.Client
}

// setHeaders adds the necessary headers for authenticated requests.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("X-MitsContextKey", c.token)
	req.Header.Set("User-Agent", "melcloud-go") // Keep consistent UA
	req.Header.Set("Accept", "application/json")
	// Add other headers from _headers in python if needed
}

// Login authenticates with MELCloud using email and password from environment variables
// and returns a new Client.
func Login() (*Client, error) {
	email := os.Getenv("MELCLOUD_EMAIL")
	password := os.Getenv("MELCLOUD_PASSWORD")

	if email == "" || password == "" {
		return nil, fmt.Errorf("MELCLOUD_EMAIL and MELCLOUD_PASSWORD environment variables must be set")
	}

	body := map[string]interface{}{
		"Email":           email,
		"Password":        password,
		"Language":        0, // Assuming default language
		"AppVersion":      appVersion,
		"Persist":         true,
		"CaptchaResponse": nil,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request body: %w", err)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/Login/ClientLogin", baseURL), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "melcloud-go") // Simple user agent

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			return nil, fmt.Errorf("login failed with status code: %d, details: %v", resp.StatusCode, errBody)
		}
		return nil, fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

	var loginResponse LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}

	if loginResponse.ErrorId != nil || loginResponse.ErrorCode != nil {
		return nil, fmt.Errorf("login API returned an error: ID=%v, Code=%v", loginResponse.ErrorId, loginResponse.ErrorCode)
	}

	if loginResponse.LoginData.ContextKey == "" {
		return nil, fmt.Errorf("login response did not contain ContextKey")
	}

	client := &Client{
		token:      loginResponse.LoginData.ContextKey,
		httpClient: httpClient,
	}

	return client, nil
}

// ListDevices fetches all devices associated with the account.
func (c *Client) ListDevices() ([]Device, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/User/ListDevices", baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list devices request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list devices request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			return nil, fmt.Errorf("list devices failed with status code: %d, details: %v", resp.StatusCode, errBody)
		}
		return nil, fmt.Errorf("list devices failed with status code: %d", resp.StatusCode)
	}

	var buildings []Building
	if err := json.NewDecoder(resp.Body).Decode(&buildings); err != nil {
		return nil, fmt.Errorf("failed to decode list devices response: %w", err)
	}

	// Extract devices from the nested structure, similar to pymelcloud
	var allDevices []Device
	visited := make(map[int]struct{}) // Use map for efficient lookup

	for _, building := range buildings {
		structure := building.Structure
		for _, device := range structure.Devices {
			if _, found := visited[device.DeviceID]; !found {
				allDevices = append(allDevices, device)
				visited[device.DeviceID] = struct{}{}
			}
		}
		for _, area := range structure.Areas {
			for _, device := range area.Devices {
				if _, found := visited[device.DeviceID]; !found {
					allDevices = append(allDevices, device)
					visited[device.DeviceID] = struct{}{}
				}
			}
		}
		for _, floor := range structure.Floors {
			for _, device := range floor.Devices {
				if _, found := visited[device.DeviceID]; !found {
					allDevices = append(allDevices, device)
					visited[device.DeviceID] = struct{}{}
				}
			}
			for _, area := range floor.Areas {
				for _, device := range area.Devices {
					if _, found := visited[device.DeviceID]; !found {
						allDevices = append(allDevices, device)
						visited[device.DeviceID] = struct{}{}
					}
				}
			}
		}
	}

	return allDevices, nil
}

// GetDeviceState fetches the current state of a specific device.
// Note: MELCloud rate limits this endpoint. Avoid calling too frequently.
func (c *Client) GetDeviceState(deviceID, buildingID int) (*AtaDeviceState, error) {
	url := fmt.Sprintf("%s/Device/Get?id=%d&buildingID=%d", baseURL, deviceID, buildingID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get device state request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get device state request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			return nil, fmt.Errorf("get device state failed for device %d (building %d) with status code: %d, details: %v", deviceID, buildingID, resp.StatusCode, errBody)
		}
		return nil, fmt.Errorf("get device state failed for device %d (building %d) with status code: %d", deviceID, buildingID, resp.StatusCode)
	}

	var state AtaDeviceState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode get device state response for device %d: %w", deviceID, err)
	}

	// Add back BuildingID as it's not always present in the response
	state.BuildingID = buildingID

	return &state, nil
}

// SetDeviceState sends updated state information to a device.
// The input `state` should be a modified version of a previously fetched state.
// It *must* have the correct `EffectiveFlags` and `HasPendingCommand` set.
func (c *Client) SetDeviceState(state AtaDeviceState) (*AtaDeviceState, error) {
	// Ensure crucial fields for setting state are present/set
	if state.EffectiveFlags == 0 {
		return nil, fmt.Errorf("SetDeviceState requires EffectiveFlags to be set to indicate changes")
	}
	state.HasPendingCommand = true // Must be true when sending commands

	// Determine the correct API endpoint based on DeviceType
	var setURL string
	switch state.DeviceType {
	case 0: // ATA (Air-to-Air)
		setURL = fmt.Sprintf("%s/Device/SetAta", baseURL)
	// TODO: Add cases for ATW (1) and ERV (3) if needed later
	default:
		return nil, fmt.Errorf("unsupported device type for SetDeviceState: %d", state.DeviceType)
	}

	jsonBody, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal set device state request body: %w", err)
	}

	req, err := http.NewRequest("POST", setURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create set device state request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute set device state request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			return nil, fmt.Errorf("set device state failed for device %d with status code: %d, details: %v", state.DeviceID, resp.StatusCode, errBody)
		}
		return nil, fmt.Errorf("set device state failed for device %d with status code: %d", state.DeviceID, resp.StatusCode)
	}

	// Parse the response, which should be the updated state
	var updatedState AtaDeviceState
	if err := json.NewDecoder(resp.Body).Decode(&updatedState); err != nil {
		return nil, fmt.Errorf("failed to decode set device state response for device %d: %w", state.DeviceID, err)
	}

	// Add back BuildingID as it's not always present in the response
	// (Use the ID from the input state as it won't change)
	updatedState.BuildingID = state.BuildingID

	return &updatedState, nil
}

