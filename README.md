# colex

*WARNING: This project is a learning exercise. Do not assume your silo'ed processes are 'secure' in any way! There are both known security flaws and those due to my ignorance.*

# colexd / colex-cli

Colexd is a container host, which gets its instructions from the network (through use of colex-cli).

## server quickstart

```shell
mkdir server
cd server
export GOPATH=`pwd`
go get github.com/twitchyliquid64/colex
go build github.com/twitchyliquid64/colex/colexd
cp github.com/twitchyliquid64/colex/busybox.tar ./ # Use busybox-arm.tar if running on ARM (eg: raspberry pi).
sudo ./colexd --addr localhost:8080 --ip-pool 10.21.0.1/24 # colexd must run as root for now, cuz containers + cbf
```

## client quickstart

```shell
export GOPATH=`pwd`
go get github.com/twitchyliquid64/colex
go build github.com/twitchyliquid64/colex/colexd/colex-cli
```

## usage

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

## Setting up the server properly

Now that you've had a taste, lets spend some time to write a proper config and enforce security properties.

### 1. Server configuration file + TLS keys

Create two files in your server directory: `config.hcl`, and `authorized-users`.

Write the following into `config.hcl`:

```hcl
name = "my host"
listener = ":8080"
address_pool = "10.21.0.1/24"

authentication {
  mode = "certs-file"
  certs_file = "authorized-users"
}
```

Start the server (using the command below) and stop it again once it has started listening. It would have spat out a block of text like this:

```
Add this section to your configuration file:
transport_security {
  key_source = "embedded"
  embedded_cert = "..."
  embedded_key = "..."
}
```

Do what it says, and copy paste the transport_security block into the bottom of `config.hcl`. From now on, the server will use the generated TLS certificate and key.

You can finally start the server for good, using `sudo ./colexd config.hcl`.

### 2. Client enrollment

> TL;DR: Run `colex-cli --serv <addr> enroll` and put in the enrollment key printed out in the servers log.

You may have noticed the `authentication` block in `config.hcl`. This tells `colexd` how to verify commands coming from `colex-cli` are legitimate. By default, `mode` is set to `open`, meaning anyone with network connectivity to colexd can run commands and manage silos.

By setting the mode to `certs-file` and pointing to another file with `certs_file`, colexd will check the file to verify the certificate presented by `colex-cli`, as well as
add new entries to the file for successful enrollments.

To enroll a client:

1. Start the server if you havent already.
2. Copy paste the enrollment key (not including quotes) - this should have been printed soon after the server started.
3. Run `colex-cli --serv <addr> enroll` and paste the enrollment key when prompted. But be quick, enrollment is only available by default for 35 seconds! (configurable with `blind_enrollment_seconds` option).

## How to I get other base images?

Docker has a great selection of images, but they need to be exported to a flat tarball before colex can use them.

**EG: busybox**

```shell
export IMGNAME="busybox"

docker pull $IMGNAME
docker create $IMGNAME containerz
docker export $(docker ps --latest --quiet) > "${IMGNAME}.tar"
docker rm $(docker ps --latest --quiet)
docker image rm $IMGNAME
```

## Configuration file reference

 - [Server configuration file](https://github.com/twitchyliquid64/colex/blob/master/server%20config%20reference.MD)
 - [Silo configuration file](https://github.com/twitchyliquid64/colex/blob/master/silo%20configuration%20reference.MD)

# credit

This was exploratory / learning exercise, made with heavy reference to https://www.infoq.com/articles/build-a-container-golang &
https://medium.com/@teddyking/linux-namespaces-850489d3ccf & https://blog.scottlowe.org/2013/09/04/introducing-linux-network-namespaces/ & https://google.com.
