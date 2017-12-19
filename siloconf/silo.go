package siloconf

// SiloFile represents the structure of a siloconf file.
type SiloFile struct {
	Silos []*Silo `hcl:"silo"`
}

// Silo represents configuration options for a silo in a silo configuration file.
type Silo struct {
	Name, Class string
	Tags        []string

	// Base indicates what root filesystem should be used to construct the
	// root filesystem when constructing the silo. The following formats are
	// supported:
	// img://<img-name> - Uses the image with image-name as per configuration on host.
	// TODO: Support - http[s]://... to download a zip/tar
	Base string

	Network Network

	Files map[string]File `hcl:"file"`
}

// File represents configuration for a silo file resource to be put into the
// environment
type File struct {
	Path     string `hcl:"path"`
	SiloPath string `hcl:"silo_path"`
}

// Network represents silo network configuration.
type Network struct {
	InternetAccess bool `hcl:"internet_access"`
	Nameservers    []string
	Hosts          map[string]string
}
