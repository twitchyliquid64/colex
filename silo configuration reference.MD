# Silo configuration file reference

Colex configuration files follow the syntax of Hashicorp Configuration Files.

## Example

```hcl

Silo "s" {
  name = "test-sh"

  binary {
    path = "/bin/sh"
    args = ["/script.sh"]
    env = ["WELP=y"]
  }

  file "main script" {
    path = "test-script.sh"
    silo_path = "/script.sh"
  }

  files "things directory" {
    path = "things/"
    silo_path = "/welp"
  }

  network {
    internet_access = true
  }
}


```

**Explanation**

* There is one silo in this file, called `test-sh`.
* The *binary* section specifies what should be run in the silo. *path* is the path to the binary to be invoked, *args* is it's list of arguments, and *env* allows you to specify environment variables.
* Each *file* section allows you to specify files on your current system which will be dropped in the silo. The 'main script' block references a file on your system `test-script.sh` which will be dropped in the silo at `/script.sh`. `test-script.sh` well be read from the filesystem when you run `colex-cli up`, and for relative paths like in this example, will source the file relative to the current directory.
* Each *files* section allows you to specify directories/globs on your current system which will be dropped in the silo. The 'things directory' block references a folder `things/` on your system which will be dropped in the silo at `/welp`.
* The *network* section lets you configure networking within the silo. If *internet_access* is set to true, iptables MASQUERADE rules + routing table entries will exist to allow the silo to access the internet through the host. You can also specify *nameservers* if you want to override default DNS resolvers, but be aware that will break any colex-specific DNS resolution (.silo addresses). You can also specify *hosts* if you want to drop any name -> IP combinations into `/etc/hosts` in that container.


## Detailed reference

TODO
