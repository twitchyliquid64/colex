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
* Each *file* block allows you to specify files on your current system which will be dropped in the silo. The 'main script' block references a file on your system `test-script.sh` which will be dropped in the silo at `/script.sh`. `test-script.sh` well be read from the filesystem when you run `colex-cli up`, and for relative paths like in this example, will source the file relative to the current directory.
* Each *files* block allows you to specify directories/globs on your current system which will be dropped in the silo. The 'things directory' block references a folder `things/` on your system which will be dropped in the silo at `/welp`.
* The *network* section lets you configure networking within the silo. If *internet_access* is set to true, iptables MASQUERADE rules + routing table entries will exist to allow the silo to access the internet through the host. See below for more options.


## Detailed reference

**Top-level schema:**

| Name                      | Description                                                                                                                  |
| -------------------       |:-----------------------------------------------------------------------------------------------------------------------------|
| *name*                    | Name of the silo. Must be unique on the instance of colexd, used when controlling or talking to the silo.                    |
| *class*                   | You can set this to a non-unique string to identify similar silos.                                                           |
| *tags*                    | List of tags (queryable) to be associated with the instance.                                                                 |
| *base*                    | Specifies which base filesystem should be installed for this silo (default = `img://busybox`). If not the default, this must reference a custom image specified in *colexd*'s config file. |
| *network*                 | This block specifies options for configuring the network.                                                                    |
| *binary*                  | This block specifies the binary to be invoked in the silo, along with it's arguments and environment variables.              |
| *file* blocks             | Each *file* block specifies a single file which should be copied into the silo.                                              |
| *files* blocks            | Each *files* block specifies a folder containing files/folders which should be copied into the silo.                         |
| *grants*                  | This block specifies additional runtime permissions.                                                                         |


**Network schema:**

| Name                      | Description                                                                                                                          |
| -------------------       |:-------------------------------------------------------------------------------------------------------------------------------------|
| *internet_access*         | This boolean specifies whether NAT routing should be configured to provide the silo with internet access (via the host).             |
| *nameservers*             | List of DNS servers to override defaults. Be aware that will break any colex-specific DNS resolution (`<silo-name>.silo` addresses). |
| *hosts*                   | This block allows you to specify custom host -> IP mappings, which will be added to `/etc/hosts` in the silo.                        |


**Binary schema:**

| Name                      | Description                                                                                                                          |
| -------------------       |:-------------------------------------------------------------------------------------------------------------------------------------|
| *path*                    | This string specifies the full path to the binary that should be run inside the container.                                           |
| *args*                    | List of arguments the binary inside the container should be invoked with.                                                            |
| *env*                     | List of additional environment variables the binary should have.                                                                     |

**Bind schema:**

| Name                      | Description                                                                                                                          |
| -------------------       |:-------------------------------------------------------------------------------------------------------------------------------------|
| *id*                      | Identifies which bind path to use. Should match an entry in the servers configuration with the same id.                              |
| *path*                    | This string specifies the full path where the file/dir should be mounted, within the silo.                                           |


## Additional environment variables

The following environment variables are added by default and cannot be disabled (file a FR if you actually need to).

 * `METADATA_ENDPOINT` - Specifies the host:port combination the metadata server can be reached on.
 * `CLASS` - Exposes the class specified in the silo config.
 * `PS1` - Set to `\u@\h:\w> `.

## Additional permissions

By default, what silo's can do at runtime is restricted. Silos can get additional permissions by requesting them using the `grants` block.

The following example gives the silo permission to read all silo's metadata and statistics using the `/silo/{SID}/(meta|netstats)`:

```hcl
grants = {
  query_silos = true
}
```
