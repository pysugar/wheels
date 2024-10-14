
```bash
curl --proxy http://192.168.1.5:8000 http://192.168.1.5:8080 -v
*   Trying 192.168.1.5:8000...
* Connected to 192.168.1.5 (192.168.1.5) port 8000
> GET http://192.168.1.5:8080/ HTTP/1.1
> Host: 192.168.1.5:8080
> User-Agent: curl/8.5.0
> Accept: */*
> Proxy-Connection: Keep-Alive
>
< HTTP/1.1 200 OK
< Content-Type: text/plain; charset=utf-8
< Date: Mon, 14 Oct 2024 17:07:11 GMT
< Content-Length: 224
<
HTTP Request Information:
Client IP: 192.168.1.5
Server IP: ::1
Server Version: 1.0.0
Request Method: GET
Request URL: /
HTTP Protocol: HTTP/1.1
Headers:
  User-Agent: curl/8.5.0
  Accept: */*
  Proxy-Connection: Keep-Alive
* Connection #0 to host 192.168.1.5 left intact
```

```bash
$ curl -k --proxy http://192.168.1.5:8000 https://192.168.1.5:8443 -v
*   Trying 192.168.1.5:8000...
* Connected to 192.168.1.5 (192.168.1.5) port 8000
* CONNECT tunnel: HTTP/1.1 negotiated
* allocate connect buffer
* Establish HTTP proxy tunnel to 192.168.1.5:8443
> CONNECT 192.168.1.5:8443 HTTP/1.1
> Host: 192.168.1.5:8443
> User-Agent: curl/8.5.0
> Proxy-Connection: Keep-Alive
>
< HTTP/1.1 200 Connection Established
<
* CONNECT phase completed
* CONNECT tunnel established, response 200
* ALPN: curl offers h2,http/1.1
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* TLSv1.3 (IN), TLS handshake, Server hello (2):
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
* TLSv1.3 (IN), TLS handshake, Certificate (11):
* TLSv1.3 (IN), TLS handshake, CERT verify (15):
* TLSv1.3 (IN), TLS handshake, Finished (20):
* TLSv1.3 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.3 (OUT), TLS handshake, Finished (20):
* SSL connection using TLSv1.3 / TLS_AES_128_GCM_SHA256 / X25519 / RSASSA-PSS
* ALPN: server accepted h2
* Server certificate:
*  subject: C=AU; ST=Some-State; O=Internet Widgits Pty Ltd
*  start date: Oct 14 17:01:11 2024 GMT
*  expire date: Oct 14 17:01:11 2025 GMT
*  issuer: C=AU; ST=Some-State; O=Internet Widgits Pty Ltd
*  SSL certificate verify result: self-signed certificate (18), continuing anyway.
*   Certificate level 0: Public key type RSA (4096/152 Bits/secBits), signed using sha256WithRSAEncryption
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* using HTTP/2
* [HTTP/2] [1] OPENED stream for https://192.168.1.5:8443/
* [HTTP/2] [1] [:method: GET]
* [HTTP/2] [1] [:scheme: https]
* [HTTP/2] [1] [:authority: 192.168.1.5:8443]
* [HTTP/2] [1] [:path: /]
* [HTTP/2] [1] [user-agent: curl/8.5.0]
* [HTTP/2] [1] [accept: */*]
> GET / HTTP/2
> Host: 192.168.1.5:8443
> User-Agent: curl/8.5.0
> Accept: */*
>
< HTTP/2 200
< content-type: text/plain; charset=utf-8
< content-length: 193
< date: Mon, 14 Oct 2024 17:02:50 GMT
<
HTTP Request Information:
Client IP: 192.168.1.5
Server IP: ::1
Server Version: 1.0.0
Request Method: GET
Request URL: /
HTTP Protocol: HTTP/2.0
Headers:
  User-Agent: curl/8.5.0
  Accept: */*
* Connection #0 to host 192.168.1.5 left intact
```