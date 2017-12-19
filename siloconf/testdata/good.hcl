
Silo "hi-silo" {
  name = "hi"
  class = "go-bin"

  base = "img://busybox"

  network {
    internet_access = true
    nameservers = ["8.8.8.8"]
    hosts = {
      b4master = "192.168.54.1"
    }
  }

  binary {
    path = "/bin/ls"
  }

  tags = ["FE"]
}

Silo "silo2" {
  name = "welp"

  binary {
    path = "/lister"
  }

  file "binary" {
   path = "/bin/ls"
   silo_path = "/lister"
  }
}
