# colex

*WARNING: This project is a learning exercise. Do not assume your silo'ed processes are 'secure' in any way! There are both known security flaws and those due to my ignorance.*

# build
```shell
go get github.com/twitchyliquid64/colex
go build github.com/twitchyliquid64/colex/colex-exec
cp ./ $GOPATH/src/github.com/twitchyliquid64/colex/busybox.tar # busybox.tar must be in your working directory
```

# use

Create a silo with the busybox environment, and get a shell there:
```shell
./colex-exec --baseline-env
```

Run `ls` instead of sh in the busybox environment:
```shell
./colex-exec --baseline-env --cmd /bin/ls
```


Create a silo with your own root FS (not busybox) and get a shell there:
```shell
./colex-exec --root_fs /mah/shitty/root/fs/base/dir
```

# credit

This was exploratory / learning exercise, made with heavy reference to https://www.infoq.com/articles/build-a-container-golang &
https://medium.com/@teddyking/linux-namespaces-850489d3ccf & google.com.
