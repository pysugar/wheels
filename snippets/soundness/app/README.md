
### 构建适用于 Linux x86_64 的可执行文件

```bash
$ GOOS=linux GOARCH=amd64 go build -o agent main.go
```

### 准备部署文件

```bash
$ cat <<EOF | sudo tee /opt/etc/agent.env > /dev/null
BROKER_URL=http://192.168.1.5:5000
AGENT_NAME=node_8
HEARTBEAT_INTERVAL=60
STATUS_PATH=/openapi
FILE_PATH=/tmp/agent-status.json
EOF

$ sudo chmod 644 /opt/etc/agent.env

$ cat <<EOF | sudo tee /etc/systemd/system/agent.service > /dev/null
[Unit]
Description=Agent Program
After=network.target

[Service]
ExecStart=/opt/bin/agent
WorkingDirectory=/tmp
Restart=always
EnvironmentFile=/opt/etc/agent.env
User=root
Group=sudo
SyslogIdentifier=znt_agent

[Install]
WantedBy=multi-user.target
EOF
```

### 启动程序

```bash
$ sudo systemctl daemon-reload

$ sudo systemctl start agent.service
$ sudo systemctl enable agent.service

$ sudo systemctl status agent.service
```