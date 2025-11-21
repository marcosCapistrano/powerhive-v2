package stock

import "errors"

var (
	// ErrNotStockFirmware indicates the host is not running stock Bitmain firmware.
	ErrNotStockFirmware = errors.New("host is not running stock Bitmain firmware")

	// ErrAuthenticationFailed indicates authentication failed.
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrConnectionFailed indicates connection to the host failed.
	ErrConnectionFailed = errors.New("connection failed")
)
