

```bash
cd cert

# 生成私钥和 CSR
openssl req -new -nodes -out server.csr -newkey rsa:2048 -keyout server.key -config server_cert.cnf

# 生成自签名证书
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt -extensions v3_req -extfile server_cert.cnf

# 验证证书
openssl x509 -in server.crt -text -noout | grep -A1 "Subject Alternative Name"
```

