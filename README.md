# colex

*WARNING: This project is a learning exercise. Do not assume your silo'ed processes are 'secure' in any way! There are both known security flaws and those due to my ignorance.*

# colexd / colex-cli

Colexd is a container host, which gets its instructions from the network (through use of colex-cli).

## setup server

```shell
mkdir server
cd server
export GOPATH=`pwd`
go get github.com/twitchyliquid64/colex
go build github.com/twitchyliquid64/colex/colexd
sudo ./colexd --addr localhost:8080 # colexd must run as root for now, cuz containers + cbf
```

## setup client

```shell
export GOPATH=`pwd`
go get github.com/twitchyliquid64/colex
go build github.com/twitchyliquid64/colex/colexd/colex-cli
```

## starting a silo

Given a silo configuration like:

```hcl
Silo "s" {
  name = "test-sh"

  binary {
    path = "/bin/sh"
    args = ["/script.sh"]
  }

  file "main script" {
    path = "test-script.sh"
    silo_path = "/script.sh"
  }

  network {
    internet_access = true
  }
}
```

Explanation:

 * The silo will be called `test-sh`. Names are unique on each host, so if you `colex-cli up` again, the existing silo will be stopped and replaced.
 * The `binary` section specifies what should be run in the container. `path` is the path to the binary to be invoked, `args` is it's list of arguments, and `env` allows you to specify environment variables.
 * Each `file` section allows you to specify files on your current system which will be dropped in the silo. The `main script` block references a file on your system `test-script.sh` which will be dropped at `/script.sh`.
 * The `network` section lets you configure networking within the silo. If `internet_access` is set to true, iptables MASQUERADE rules + routing table entries will exist to allow the silo to access the internet through the host.

To Start:

```shell
./colex-cli --serv <colexd address> <silo-file> up

|-----------|----------|-------|---------------------------|
|   SILO    |    ID    | STATE |        INTERFACES         |
|-----------|----------|-------|---------------------------|
| test-sh   | 3938a44e | UP    | v0-3938a44es (10.69.69.2) |
|-----------|----------|-------|---------------------------|
```

To stop:

```shell
./colex-cli --serv <colexd address> <silo-file> down
```

To see whats running:

```shell
./colex-cli --serv localhost:8080 list
|-----------|--------------|-------|------|------------|
|   NAME    |    ID ()     | CLASS | TAGS | ADDRESSES  |
|-----------|--------------|-------|------|------------|
| test-ping | 7d462601 (0) |       |      | 10.69.69.2 |
|-----------|--------------|-------|------|------------|
```

# credit

This was exploratory / learning exercise, made with heavy reference to https://www.infoq.com/articles/build-a-container-golang &
https://medium.com/@teddyking/linux-namespaces-850489d3ccf & https://blog.scottlowe.org/2013/09/04/introducing-linux-network-namespaces/ & https://google.com.
