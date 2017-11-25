# colex

*WARNING: This project is a learning exercise. Do not assume your silo'ed processes are 'secure' in any way! There are both known security flaws and those due to my ignorance.*

# build
```shell
go get github.com/twitchyliquid64/colex
go build github.com/twitchyliquid64/colex/colex-test
cp ./ $GOPATH/src/github.com/twitchyliquid64/colex/busybox.tar # busybox.tar must be in your working directory
```

# use the test (my early learning tool)

Create a silo with the busybox environment, and get a shell there:
```shell
./colex-test --baseline-env
```

Run `ls` instead of sh in the busybox environment:
```shell
./colex-test --baseline-env --cmd /bin/ls
```


Create a silo with your own root FS (not busybox) and get a shell there:
```shell
./colex-test --root_fs /mah/shitty/root/fs/base/dir
```

# Notes


Getting internet working inside the silo:

```shell
sudo iptables -t nat -A POSTROUTING -m physdev --physdev-in <VETH-PAIR-HOST-SIDE> -j MASQUERADE
```

# credit

This was exploratory / learning exercise, made with heavy reference to https://www.infoq.com/articles/build-a-container-golang &
https://medium.com/@teddyking/linux-namespaces-850489d3ccf & https://blog.scottlowe.org/2013/09/04/introducing-linux-network-namespaces/ & https://google.com.
