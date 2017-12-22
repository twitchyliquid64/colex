package wire

import "time"

// EnableEnrollResponse is sent in response to a successful enable-enroll RPC.
type EnableEnrollResponse struct {
	DisablesAt time.Time
	Code       string
}
