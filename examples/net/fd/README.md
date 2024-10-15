

## server

```bash
$ cd server
# 交叉编译命令
$ GOOS=linux GOARCH=amd64 go build -o fdserver
# 静态链接编译
$ GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o fdserver

$ file fdserver

$ scp fdserver ubuntu-pn51.local:/tmp
```

## client

```bash
$ cd client
$ GOOS=linux GOARCH=amd64 go build -o fdclient
```

## all in one

```bash
$ GOOS=linux GOARCH=amd64 go build -o fdclient ./client
$ GOOS=linux GOARCH=amd64 go build -o fdserver ./server

$ scp fdserver ubuntu-pn51.local:/tmp
$ scp fdclient ubuntu-pn51.local:/tmp
```


```bash
$ ./fdserver

$ ls -lah /proc/2302/fd/
$ cat /proc/net/tcp
$ ss -ltn
```