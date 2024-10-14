
### lsof

#### 查看某个进程打开的文件和端口

```bash
$ lsof -p <PID>
```

#### 查找某个进程（使用名称）所打开的文件或端口

```bash
$ lsof -c <进程名称>
```

#### 只查看进程打开的网络连接（端口）

```bash
$ lsof -i -a -p <PID>
```

#### 查看特定端口被哪个进程占用

```bash
$ lsof -i :8080
```

### netstat

```bash
# 查看所有监听端口
$ netstat -an | grep LISTEN

# 查看具体端口的详细信息
$ netstat -an | grep 8080
```
