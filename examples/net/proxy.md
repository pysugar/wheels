




```bash
$ cd httpserver
$ go run main.go

$ cd httpproxy
$ go run main.go

$ cd httpserver
# 创建自签名证书和私钥
$ openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

```bash
$ curl -x http://192.168.1.5:8000 http://192.168.1.5:8080 -v
$ curl --proxy http://192.168.1.5:8000 http://192.168.1.5:8080 -v

$ curl -k --proxy http://192.168.1.5:8000 https://192.168.1.5:8443 -v
```

