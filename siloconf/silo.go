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

	Binary Binary

	Resources Resources

	Grant map[string]bool `hcl:"grants"`

	Files     map[string]File `hcl:"file"`
	FileBalls map[string]File `hcl:"files"`
	Binds     map[string]Bind `hcl:"bind"`
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

// Binary represents the initial invocation details of the silo.
type Binary struct {
	Path string   `hcl:"path"`
	Env  []string `hcl:"env"`
	Args []string `hcl:"args"`
}

// Bind indicates a path on the host should be present in the silo.
type Bind struct {
	ID   string `hcl:"id"`
	Path string `hcl:"path"`
}

// Resources specifies resource constraints for the silo.
type Resources struct {
	CPUPercent int `hcl:"cpu_percent"`
}
