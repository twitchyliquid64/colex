name = "test-host"
listener = "localhost:8080"
address_pool = "10.21.0.1/24"

image "base" {
  type = "tarball"
  name = "busybox-custom"
  path = "busybox.tar"
}
