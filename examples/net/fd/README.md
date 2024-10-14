


```bash
# 交叉编译命令
$ GOOS=linux GOARCH=amd64 go build -o fdtest
# 静态链接编译
$ GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o fdtest

$ file fdtest

$ scp fdtest ubuntu-pn51.local:/tmp
```

```bash
$ ./fdtest

$ ls -lah /proc/2302/fd/
$ cat /proc/net/tcp
$ ss -ltn
```