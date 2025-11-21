package miner

// Info contains basic miner information.
// This is a firmware-agnostic representation of miner details.
type Info struct {
	// Miner is the miner name (e.g., "Antminer S19").
	Miner string

	// Model is the model identifier (e.g., "s19").
	Model string

	// Series is the product series (e.g., "x19").
	Series string

	// Firmware is the firmware name (e.g., "Vnish").
	Firmware string

	// FirmwareVersion is the firmware version (e.g., "1.2.6").
	FirmwareVersion string

	// Algorithm is the mining algorithm (e.g., "sha256d").
	Algorithm string

	// IP is the miner's IP address.
	IP string

	// MAC is the miner's MAC address.
	MAC string

	// Hostname is the miner's hostname.
	Hostname string
}

// Status contains the miner's operational status.
type Status struct {
	// State is the current miner state (e.g., "running", "stopped", "failure").
	State string

	// Description is a human-readable status description.
	Description string

	// FailureCode is the error code if in failure state.
	FailureCode int
}

// FirmwareType represents different firmware types.
type FirmwareType string

const (
	FirmwareVNish   FirmwareType = "vnish"
	FirmwareStock   FirmwareType = "stock"
	FirmwareBraiins FirmwareType = "braiins"
	FirmwareUnknown FirmwareType = "unknown"
)
