# Server configuration file reference

Colex configuration files follow the syntax of Hashicorp Configuration Files.

## Example

```hcl
name = "test-host"
listener = ":8080"
address_pool = "10.21.0.1/24"

image "base" {
  type = "tarball"
  name = "busybox-custom"
  path = "busybox-custom.tar"
}

transport_security {
  key_source = "embedded"
  embedded_cert = "-----BEGIN CERTIFICATE-----\n{SNIPPED}\n-----END CERTIFICATE-----\n"
  embedded_key = "-----BEGIN RSA PRIVATE KEY-----\n{SNIPPED}\n-----END RSA PRIVATE KEY-----\n"
}

authentication {
  mode = "certs-file"
  certs_file = "authorized-users"
}

```

**Explanation**

 * The host identifies itself as 'test-host'.
 * The host listens on port 8080 (all interfaces) for commands from `colex-cli`.
 * Silo interfaces (as well as bridge interfaces) are assigned IP addresses from 10.21.0.1 to 10.21.0.255. When a silo is freed, the addresses corresponding to it's interfaces are re-used.
 * An additional image is made available, called *busybox-custom*. Silos can use this by having `base = "img://busybox-custom"` in the silo config. The referenced tarball is unpacked to provide the base filesystem for these silos. In this case `path` is a relative path, so it is sourced relative to the current directory of `colexd`.
 * The *transport_security* section details how the network service (which `colex-cli` connects to) is encrypted. If *key_source* is set to *embedded*, *embedded_cert* and *embedded_key* have to contain a PEM-encoded cert/private-key, with the newlines escaped to `\n`. This cert/key is then used as a TLS server certificate.
 * The *authentication* section specifies how RPCs/commands from `colex-cli` are authenticated - in other words, what criteria they need to meet before colex will accept the command. With
 *mode* set to *certs-file*, no command will be accepted from `colex-cli` unless the TLS public key used by `colex-cli` is in the file referenced by *certs_file*.


## Detailed reference

**Top-level schema:**

| Name                 | Description                                                                                                          |
| -------------------  |:---------------------------------------------------------------------------------------------------------------------|
| *name*               | Name of the host. Mostly unused, except for display to the user.                                                     |
| *listener*           | Host and port on which the service listens. Format: [host]:port.                                                     |
| *address_pool*       | CIDR representing the starting IP and subnet from which silos are allocated IPs.                                     |
| *image*              | Image sections specify additional base images which are available.                                                   |
| *transport_security* | TLS security options for the service.                                                                                |
| *authentication*     | Details how RPCs/command from `colex-cli` are authorized.                                                            |