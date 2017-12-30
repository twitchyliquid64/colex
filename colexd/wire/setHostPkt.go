package wire

// SetHostRequest represents a request to set a hostname - IP pair which will resolve for silos.
type SetHostRequest struct {
	Host string
	IP   string
}
