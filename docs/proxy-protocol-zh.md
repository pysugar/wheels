2020/03/05                                                       

此文档详细描述了用于在多层代理之间传递连接信息的协议。
它定义了两个版本的PROXY协议：版本1（文本格式）和版本2（二进制格式）。
该协议帮助传递原始TCP连接参数，例如源和目标地址。
版本2增加了TLV（类型-长度-值）扩展，以提供更多的元数据，例如ALPN和SSL相关信息。
通过使用PROXY协议，代理可以更好地在多层架构中保持透明性，支持多层代理集成、IPv4和IPv6兼容，以及提高安全性。

# PROXY协议 (版本1和版本2)

## 摘要

PROXY协议提供了一种安全传输连接信息（例如客户端地址）的方法，适用于多层NAT或TCP代理。这种协议的设计目标是对现有组件的修改最小化，同时尽量减少因处理传输信息而引起的性能影响。

## 修订历史

- **2010/10/29** - 初版发布
- **2011/03/20** - 更新：实现和安全性考虑
- **2012/06/21** - 添加对二进制格式的支持
- **2012/11/19** - 最终审阅与修正
- **2014/05/18** - 修改并扩展了PROXY协议版本2
- **2014/06/11** - 修复示例代码以考虑`ver+cmd`的合并
- **2014/06/14** - 修复v2头部检查的示例代码，并更新“Forwarded”规范
- **2014/07/12** - 更新实现列表（添加了Squid）
- **2015/05/02** - 更新实现列表以及TLV附加信息的格式
- **2017/03/10** - 添加校验和，无操作符以及更多与SSL相关的TLV类型，保留TLV类型范围，添加TLV文档，明确字符串编码。感谢Andriy Palamarchuk (Amazon.com)的贡献。
- **2020/03/05** - 添加唯一ID的TLV类型（Tim Düsterhus贡献）

## 1. 背景

通过代理中继TCP连接通常会导致原始TCP连接参数（如源地址、目标地址、端口等）丢失。某些协议稍微更容易传输这些信息。
例如，SMTP中，Postfix作者提出的XCLIENT协议[1]得到了广泛的采用，非常适合邮件传输。
在HTTP中，有一个名为“Forwarded”的扩展[2]，其目的是取代无处不在的“X-Forwarded-For”头，该头部包含原始源地址的信息，以及较少使用的“X-Original-To”头，它包含目标地址的信息。

但是，这些机制都需要中间设备对底层协议有一定的理解。

接着出现了新一类产品，我们称之为“哑代理（dumb proxies）”，并不是因为它们无所作为，而是因为它们处理的是与协议无关的数据。
Stunnel[3]和Stud[4]都是这样的“哑代理”的例子。它们一边处理原始TCP，另一边处理原始SSL，并且可靠地工作，而不需要知道连接上层传输的是什么协议。
在纯TCP模式下运行的HAProxy也属于这一类。

当这样的代理与另一个代理（例如haproxy）结合使用时，问题是如何使其能够处理更高级别的协议。
为Stunnel提供了一个补丁，使其能够在每个传入连接的第一个HTTP请求中插入“X-Forwarded-For”头部。
HAProxy能够在连接来自Stunnel时不再添加另一个头部，从而使其可以对服务器隐藏。

典型的架构如下：

```                          
      +--------+      HTTP                      :80 +----------+
      | client |  --------------------------------> |          |
      |        |                                    | haproxy, |
      +--------+             +---------+            |  1 or 2  |
     /        /     HTTPS    | stunnel |  HTTP  :81 | listening|
    <________/    ---------> | (server | ---------> |  ports   |
                             |  mode)  |            |          |
                             +---------+            +----------+
```

当haproxy在面向客户端一侧启用长连接（keep-alive）时就会出现问题。
Stunnel补丁只会在每个连接的第一个请求中添加“X-Forwarded-For”头，之后的请求将不会有它。
一个解决方案是改进补丁，使其支持长连接并解析所有转发的数据，无论是通过Content-Length声明的，还是使用Transfer-Encoding的，还需考虑像HEAD这样宣布数据但不传输的特殊方法等等……实际上，这就要求在Stunnel中实现完整的HTTP协议栈，这会使其变得更加复杂，可靠性下降，也就不再是适合各种用途的“哑代理”了。



在实际应用中，我们不需要对每个请求添加一个头部，因为每次传递的信息都是相同的——即客户端连接相关的信息。
因此，我们可以将这些信息缓存到haproxy中，供每个请求重复使用。
但这种方式具有风险，并且仍然只限于HTTP协议。

另一种方法是在每个连接的前面添加一个头部，报告对方连接的特征。
这种方式更易于实现，不需要对双方的协议有特定的了解，并且完全符合目的，因为我们只需要了解对方的连接端点。
对于发送方来说，执行起来很简单（连接建立后只需发送一个简短的头部），对于接收方来说，解析也很简单（只需在接受后对传入连接进行一次读取以填充地址）。
用于在代理之间传输连接信息的协议因此被称为PROXY协议。

## PROXY协议头部

### 2. PROXY协议头部

本文档中使用了一些术语，以下是它们的解释：
- **"连接发起者（connection initiator）"**：请求建立新连接的一方。
- **"连接目标（connection target）"**：接受连接请求的一方。
- **"客户端（client）"**：发起连接请求的一方。
- **"服务器（server）"**：客户端希望连接的目标。
- **"代理（proxy）"**：拦截并将连接从客户端转发到服务器的中介方。
- **"发送方（sender）"**：通过连接发送数据的一方。
- **"接收方（receiver）"**：接收来自发送方数据的一方。
- **"头部（header）"或"PROXY协议头部"**：连接发起者在连接开始时添加的连接信息块，在协议角度来看，它使发起者成为发送方。

PROXY协议的目标是将代理收集的信息填充到服务器的内部结构中，使得如果客户端直接连接到服务器而不是通过代理，服务器也能获得这些信息。
协议中传输的信息是服务器通过`getsockname()`和`getpeername()`可以获取的内容：
- 地址族（例如IPv4的AF_INET，IPv6的AF_INET6，AF_UNIX等）
- 套接字协议（如SOCK_STREAM代表TCP，SOCK_DGRAM代表UDP）
- 三层源地址和目标地址
- 四层源端口和目标端口（如果有）

与XCLIENT协议不同，PROXY协议设计之初就考虑到了扩展性限制，目的是让接收方可以快速解析。
版本1专注于保持协议的人类可读性，以便于调试，尤其是在早期采用且实现较少的时候。
版本2添加了对头部的二进制编码支持，这种编码方式比文本方式更高效，尤其是在处理IPv6地址时，因为IPv6地址以ASCII形式表示和解析的开销较大。

无论哪种情况，协议都是在每个连接开始时，由连接发起者放置一个易于解析的头部。
协议的设计有意保持无状态，不要求发送方在发送头部之前等待接收方，也不要求接收方返回任何数据。

此规范支持两种头部格式：人类可读的格式（仅在版本1中支持），以及二进制格式（仅在版本2中支持）。
两种格式都设计为无法与常见的高层协议（如HTTP、SSL/TLS、FTP或SMTP）混淆，并且接收方能够轻松区分这两种格式。

版本1的发送方**可能**只产生人类可读的头部格式。
版本2的发送方**可能**只产生二进制头部格式。
版本1的接收方**必须**至少实现人类可读的头部格式。
版本2的接收方**必须**至少实现二进制头部格式，并建议它们也实现人类可读的头部格式，以便更好地兼容和应对版本1的发送方。

两种格式都设计为适应最小的TCP段（576字节减去40字节的TCP/IP头部，即536字节），以确保在连接开始时，整个头部可以一次性发送和接收。
发送方必须确保头部一次性发送，以保持传输层的原子性。
接收方可以容忍部分头部，但也可能直接丢弃不完整的头部。
建议容忍部分头部，但在实现上可能存在限制。
由于TCP是流协议，可能会逐字节传输，因此中间设备并不强制将整个头部一次性转发。
不过，实际应用中遇到逐字节传输的风险很低，因此上述简化一般是可接受的。

接收方在接收到完整且有效的PROXY协议头部前，**不得**开始处理连接，这在SMTP、FTP或SSH等需要接收方先发送数据的协议中尤其重要。
如果未在几秒内收到协议头部（至少3秒以覆盖TCP重传），接收方可以设置短超时并决定中止连接。

接收方必须配置为只接受本规范中描述的协议，**不得**尝试猜测协议头部是否存在。
这意味着协议明确防止公共和私有访问之间共享端口，否则会造成重大安全漏洞，使不受信任的一方可以伪造其连接地址。
接收方应确保适当的访问过滤，以确保只有受信任的代理能够使用此协议。

某些代理足够智能，能够理解所传输的协议，并复用空闲的服务器连接以处理多个消息。
这种情况通常发生在HTTP中，不同客户端的请求可能通过同一连接发送。
此类代理**不得**在多路复用连接上实现此协议，因为接收方会使用PROXY头部中报告的地址作为所有转发请求的发送者地址。
事实上，这些代理不是“哑代理”，因为它们完全理解传输的协议，因此**必须**使用协议提供的工具来显示客户端地址。

### 2.1. 人类可读头部格式（版本1）

这种格式是协议版本1中指定的格式。它由一行US-ASCII文本组成，连接建立后立即发送，并在所有从发送方到接收方的数据之前附加：

- **标识协议的字符串**：`"PROXY"` （\x50 \x52 \x4F \x58 \x59）。看到该字符串意味着这是协议的第一个版本。
- **一个空格**：`" "` （\x20）。
- **表示代理的INET协议和地址族的字符串**：在版本1中，只有`"TCP4"`（IPv4上的TCP，\x54 \x43 \x50 \x34）和`"TCP6"`（IPv6上的TCP，\x54 \x43 \x50 \x36）是允许的。其他不支持或未知的协议必须报告为`"UNKNOWN"`（\x55 \x4E \x4B \x4E \x4F \x57 \x4E）。对于`"UNKNOWN"`，发送方可以省略行的其余部分，接收方必须忽略CRLF之前的任何内容。需要注意的是，早期版本的规范建议在发送健康检查时使用此项，但这可能导致某些服务器拒绝`"UNKNOWN"`关键字。因此，现在建议不要在连接预计会被接受时发送`"UNKNOWN"`，而只能在无法正确填写PROXY行时使用。
- **一个空格**：`" "` （\x20）。
- **三层源地址**：使用标准的IPv4或IPv6地址格式。IPv4地址由4个整数表示，范围为[0..255]，每个数之间用点隔开。前导零不允许，以避免与八进制数字混淆。IPv6地址由4位十六进制数字组成，每组之间用冒号隔开，支持使用双冒号替换连续的零。总共128位，地址族决定了使用的格式。
- **一个空格**：`" "` （\x20）。
- **三层目标地址**：使用与三层源地址相同的格式。
- **一个空格**：`" "` （\x20）。
- **TCP源端口**：表示为十进制整数，范围为[0..65535]。不允许前导零以避免与八进制数字混淆。
- **一个空格**：`" "` （\x20）。
- **TCP目标端口**：表示为十进制整数，范围为[0..65535]。不允许前导零以避免与八进制数字混淆。
- **CRLF序列**：`\x0D \x0A`。

接收方必须等待接收到CRLF序列后才能开始解析地址，以确保地址完整并已正确解析。
如果在前107个字符中未找到CRLF序列，接收方应声明此行为无效。
接收方可能拒绝在第一次读取操作中未包含CRLF序列的不完整行。

任何不符合协议的序列都必须被丢弃，并导致接收方中止连接。建议尽早中止连接，以便发送方有机会注意到异常并记录。

如果传输协议声明为`"UNKNOWN"`，则接收方知道发送方使用的是正确版本的PROXY协议，应接受连接并使用真实连接的参数，就像没有PROXY协议头一样。
但是，当发送方发起出站连接时，不应使用`"UNKNOWN"`协议，因为有些接收方可能会拒绝它们。
当负载均衡代理需要向服务器发送健康检查时，它应构建有效的PROXY行，并用`getsockname()/getpeername()`对填写所用地址。
需要理解的是，当发送方和接收方之间进行了源地址转换时，这样做可能是不合适的。

接收方必须支持的最大行长度（包括CRLF）为：
- **TCP/IPv4**：
  `"PROXY TCP4 255.255.255.255 255.255.255.255 65535 65535\r\n"`
  计算为：5 + 1 + 4 + 1 + 15 + 1 + 15 + 1 + 5 + 1 + 5 + 2 = 56个字符。

- **TCP/IPv6**：
  `"PROXY TCP6 ffff:f...f:ffff ffff:f...f:ffff 65535 65535\r\n"`
  计算为：5 + 1 + 4 + 1 + 39 + 1 + 39 + 1 + 5 + 1 + 5 + 2 = 104个字符。

- **未知连接（简短形式）**：
  `"PROXY UNKNOWN\r\n"`
  计算为：5 + 1 + 7 + 2 = 15个字符。

- **最坏情况下（可选字段设置为0xff）**：
  `"PROXY UNKNOWN ffff:f...f:ffff ffff:f...f:ffff 65535 65535\r\n"`
  计算为：5 + 1 + 7 + 1 + 39 + 1 + 39 + 1 + 5 + 1 + 5 + 2 = 107个字符。

因此，108字节的缓冲区足以存储整行内容及字符串处理所需的结尾空字符。

### 2.2. 二进制头部格式（版本2）

生成和解析IPv6地址的效率低下，原因在于多种可能的表示格式及处理紧凑地址格式的复杂性。
文本格式下无法指定IPv4/IPv6以外的地址族或非TCP协议。
此外，文本格式需要解析所有字符以查找结尾的CRLF，使得精确读取字节数变得困难。
最后，`UNKNOWN`地址类型因含义不明确，未必能被服务器作为有效协议接受。

因此，协议的版本2引入了一种新二进制格式，这种格式可以轻松与版本1及其他常见协议区分开来。
它的设计目的是与各种协议不兼容，避免被意外地解释为其他协议。
此外，为了提高处理效率，IPv4和IPv6地址分别以4和16字节对齐。

二进制头部格式以一个常量的12字节块作为协议签名开始：

```
\x0D \x0A \x0D \x0A \x00 \x0D \x0A \x51 \x55 \x49 \x54 \x0A
```

需要注意的是，第5个位置包含一个空字节，因此不能将其视为以空字符结尾的字符串。

第13个字节（即第13个位置）表示协议版本和命令。

- 最高4位包含协议版本。根据本规范，它始终应为`\x2`，接收方只接受此值。
- 最低4位表示命令：
    - **`\x0`**：`LOCAL`：连接由代理建立而非转发。连接端点是发送方和接收方。接收方必须接受此连接并使用真实的连接端点，忽略协议块中包含的地址族信息。
    - **`\x1`**：`PROXY`：连接是代表其他节点建立的，反映了原始连接端点。接收方必须使用协议块中提供的信息获取原始地址。
    - **其他值**：未分配，发送方不得使用。接收方必须丢弃包含意外值的连接。

第14个字节表示传输协议和地址族。最高4位包含地址族，最低4位包含传输协议。

地址族映射到原始套接字族，不一定与系统内部使用的值相匹配。可能的值包括：
- **0x0：AF_UNSPEC**：转发的连接使用未知、未指定或不受支持的协议。发送方在发送LOCAL命令或处理不支持的协议族时应使用此族。接收方可以接受连接并使用真实的端点地址，或拒绝连接。
- **0x1：AF_INET**：转发的连接使用AF_INET地址族（IPv4）。地址分别为4个字节，以网络字节序表示，后跟传输协议信息（通常是端口）。
- **0x2：AF_INET6**：转发的连接使用AF_INET6地址族（IPv6）。地址分别为16个字节，以网络字节序表示，后跟传输协议信息（通常是端口）。
- **0x3：AF_UNIX**：转发的连接使用AF_UNIX地址族（UNIX）。地址为108个字节。
- **其他值**：在协议版本2中未指定，接收方必须将其视为无效并拒绝。

第14个字节的低4位指定传输协议：
- **0x0：UNSPEC**：连接使用未知、未指定或不支持的协议。发送方在发送LOCAL命令或处理不支持的协议族时应使用此类型。接收方可以接受连接并使用真实端点地址，也可以拒绝连接，忽略地址信息。
- **0x1：STREAM**：转发的连接使用SOCK_STREAM协议（例如TCP或UNIX_STREAM）。在使用AF_INET/AF_INET6（TCP）时，地址后跟源和目标端口，各占2字节，使用网络字节序。
- **0x2：DGRAM**：转发的连接使用SOCK_DGRAM协议（例如UDP或UNIX_DGRAM）。在使用AF_INET/AF_INET6（UDP）时，地址后跟源和目标端口，各占2字节，使用网络字节序。
- **其他值**：未指定，协议版本2中不得使用，接收方应视为无效。

实际应用中，预期的协议字节包括：
- **\x00：UNSPEC**：连接使用未知、未指定或不支持的协议。LOCAL命令中，接收方必须接受连接并忽略地址信息。对于其他命令，接收方可以选择接受或拒绝。
- **\x11：TCP over IPv4**：转发的连接使用AF_INET协议族上的TCP，地址长度为2*4 + 2*2 = 12字节。
- **\x12：UDP over IPv4**：转发的连接使用AF_INET协议族上的UDP，地址长度为12字节。
- **\x21：TCP over IPv6**：转发的连接使用AF_INET6协议族上的TCP，地址长度为2*16 + 2*2 = 36字节。
- **\x22：UDP over IPv6**：转发的连接使用AF_INET6协议族上的UDP，地址长度为36字节。
- **\x31：UNIX流**：转发的连接使用AF_UNIX协议族上的SOCK_STREAM，地址长度为2*108 = 216字节。
- **\x32：UNIX数据报**：转发的连接使用AF_UNIX协议族上的SOCK_DGRAM，地址长度为216字节。

接收方必须至少实现UNSPEC协议字节（\x00）。对于不支持的有效组合，接收方应自动回退至UNSPEC模式。

第15和第16字节表示地址长度（以网络字节序表示），用于让接收方知道要跳过多少字节，即协议头部总长度始终为16 + 地址长度。
当发送方提供LOCAL连接时，不应提供任何地址，因此该字段设置为零。
接收方必须始终使用该字段来跳过适当数量的字节，不应假定LOCAL连接的值为零。

因此，16字节的版本2头部可以描述为以下结构：

```c
struct proxy_hdr_v2 {
    uint8_t sig[12];  /* hex 0D 0A 0D 0A 00 0D 0A 51 55 49 54 0A */
    uint8_t ver_cmd;  /* protocol version and command */
    uint8_t fam;      /* protocol family and address */
    uint16_t len;     /* number of following bytes part of the header */
};
```

从第17字节开始，地址按网络字节序呈现。地址顺序始终相同：
- 三层源地址（网络字节序）
- 三层目标地址（网络字节序）
- 四层源地址（如果有，端口号，网络字节序）
- 四层目标地址（如果有，端口号，网络字节序）

地址块可以直接发送或接收到以下联合体中，这样可以方便地根据地址类型转换为相关的套接字本地结构体：

```c
union proxy_addr {
    struct {        /* for TCP/UDP over IPv4, len = 12 */
        uint32_t src_addr;
        uint32_t dst_addr;
        uint16_t src_port;
        uint16_t dst_port;
    } ipv4_addr;
    struct {        /* for TCP/UDP over IPv6, len = 36 */
         uint8_t  src_addr[16];
         uint8_t  dst_addr[16];
         uint16_t src_port;
         uint16_t dst_port;
    } ipv6_addr;
    struct {        /* for AF_UNIX sockets, len = 216 */
         uint8_t src_addr[108];
         uint8_t dst_addr[108];
    } unix_addr;
};
```

发送方需要确保完整的协议头部一次性发送。由于此块总是小于MSS（最大段大小），因此在连接开始时不会被分段。
接收方也应一次性处理头部数据，并在开始解析地址之前，确保接收到完整的地址块。
如果接收方收到部分协议头部，它必须拒绝此连接。

接收方可能被配置为支持协议的版本1和版本2，识别协议版本的方式如下：
- 如果传入的数据长度为16字节或以上，且前13个字节与协议签名匹配，且第14个字节为版本2，则为版本2协议。
    ```c
    \x0D\x0A\x0D\x0A\x00\x0D\x0A\x51\x55\x49\x54\x0A\x20
    ```
- 否则，如果传入的数据长度为8字节或以上，且前5个字符为`PROXY`，则协议解析为版本1。
    ```c
    \x50\x52\x4F\x58\x59
    ```
- 否则，协议不在本规范范围内，连接应被丢弃。

在协议头部指示的地址信息之外，如果头部中包含其他字节，接收方可以选择跳过或尝试解析这些字节。
这些扩展信息以类型-长度-值（Type-Length-Value，TLV）向量的格式排列，具体描述如下：

```c
struct pp2_tlv {
    uint8_t type;
    uint8_t length_hi;
    uint8_t length_lo;
    uint8_t value[0];
};
```

接收方可以选择跳过并忽略它不感兴趣或无法理解的TLV类型。发送方可以仅生成它们选择发布的信息的TLV。

以下类型已经为`<type>`字段注册：

```c
#define PP2_TYPE_ALPN           0x01  // 应用层协议协商
#define PP2_TYPE_AUTHORITY      0x02  // 授权（通常是主机名）
#define PP2_TYPE_CRC32C         0x03  // CRC32C校验和
#define PP2_TYPE_NOOP           0x04  // 无操作符
#define PP2_TYPE_UNIQUE_ID      0x05  // 唯一ID
#define PP2_TYPE_SSL            0x20  // SSL相关信息
#define PP2_SUBTYPE_SSL_VERSION 0x21  // SSL版本
#define PP2_SUBTYPE_SSL_CN      0x22  // SSL证书中的Common Name
#define PP2_SUBTYPE_SSL_CIPHER  0x23  // SSL密码套件
#define PP2_SUBTYPE_SSL_SIG_ALG 0x24  // SSL签名算法
#define PP2_SUBTYPE_SSL_KEY_ALG 0x25  // SSL密钥算法
#define PP2_TYPE_NETNS          0x30  // 网络命名空间
```

### 2.2.1 PP2_TYPE_ALPN

应用层协议协商（ALPN）。这是一个字节序列，用于定义连接中使用的上层协议。最常见的用例是传递TLS协议的ALPN扩展的副本，具体定义见RFC7301 [9]。

### 2.2.2 PP2_TYPE_AUTHORITY

包含由客户端传递的主机名值，以UTF-8编码的字符串表示。如果客户端连接使用了TLS，则该值为RFC3546 [10]第3.1节中定义的“server_name”扩展的副本，通常称为"SNI"。在某些情况下，即使未涉及TLS，也可能在连接中提到授权信息。

### 2.2.3 PP2_TYPE_CRC32C

PP2_TYPE_CRC32C类型的值是一个32位数字，用于存储PROXY协议头部的CRC32c校验和。

当发送方在构建头部时支持校验和时，发送方必须：

- 将校验和字段初始化为‘0’。
- 按照RFC4960附录B [8]中描述的方式计算PROXY头部的CRC32c校验和。
- 将结果值放入校验和字段，并保持其余位不变。
- 
如果校验和作为PROXY头部的一部分提供，并且接收方支持校验和功能，接收方必须：

- 将接收到的CRC32c校验和值存放起来。
- 将接收到的PROXY头部中的32位校验和字段替换为全‘0’，并计算整个PROXY头部的CRC32c校验和值。
- 验证计算的CRC32c校验和是否与接收到的CRC32c校验和相同。如果不同，接收方必须将提供该头部的TCP连接视为无效。

对于无效TCP连接的默认处理方式是中止连接。

### 2.2.4 PP2_TYPE_NOOP

此类型的TLV在解析时应被忽略。其值可以为零字节或更多字节。可以用于数据填充或对齐。需要注意的是，TLV不能小于3字节，因此只能用于3个或更多字节的对齐操作。

### 2.2.5 PP2_TYPE_UNIQUE_ID

PP2_TYPE_UNIQUE_ID类型的值是由上游代理生成的最多128字节的不透明字节序列，用于唯一标识连接。

唯一ID可用于跨多个代理层轻松关联连接，而无需查找IP地址和端口号。

### 2.2.6 PP2_TYPE_SSL类型及其子类型

对于PP2_TYPE_SSL类型，其值定义如下：

```c
struct pp2_tlv_ssl {
    uint8_t  client;
    uint32_t verify;
    struct pp2_tlv sub_tlv[0];
};
```

- `<verify>`字段在客户端提供了证书且成功验证时为零，否则为非零值。
- `<client>`字段由以下位字段组成，指示哪些元素存在：

```c
#define PP2_CLIENT_SSL           0x01
#define PP2_CLIENT_CERT_CONN     0x02
#define PP2_CLIENT_CERT_SESS     0x04
```

注意，这些元素中的每一个都可能导致附加数据被添加到此TLV中，使用二级TLV封装。因此，在此字段之后可能找到多个TLV值。`pp2_tlv_ssl`的总长度将反映这一点。

- **PP2_CLIENT_SSL**标志表示客户端通过SSL/TLS连接。当此字段存在时，TLS版本的US-ASCII字符串表示形式将以TLV格式附加在字段末尾，类型为`PP2_SUBTYPE_SSL_VERSION`。
- **PP2_CLIENT_CERT_CONN**表示客户端在当前连接中提供了证书。
- **PP2_CLIENT_CERT_SESS**表示客户端在此连接所属的TLS会话中至少提供过一次证书。

二级TLV **PP2_SUBTYPE_SSL_CIPHER** 提供了所用密码套件的US-ASCII字符串名称，例如 `"ECDHE-RSA-AES128-GCM-SHA256"`。

二级TLV **PP2_SUBTYPE_SSL_SIG_ALG** 提供了前端在进行SSL/TLS传输层连接时所用签名算法的US-ASCII字符串名称，例如 `"SHA256"`。

二级TLV **PP2_SUBTYPE_SSL_KEY_ALG** 提供了前端在进行SSL/TLS传输层连接时所用密钥生成算法的US-ASCII字符串名称，例如 `"RSA2048"`。

在所有情况下，客户端证书的可分辨名称（Distinguished Name）中公共名字段（OID: 2.5.4.3）的字符串表示形式（以UTF-8编码）将使用TLV格式附加，类型为 **PP2_SUBTYPE_SSL_CN**，例如 `"example.com"`。

### 2.2.7 PP2_TYPE_NETNS类型

PP2_TYPE_NETNS类型的值定义为命名空间名称的US-ASCII字符串表示形式。

### 2.2.8 预留类型范围

以下16个类型值的范围保留用于特定应用程序的数据，PROXY协议将永不使用这些类型。如果需要更多类型值，可以考虑在TLV中扩展类型字段。

```c
#define PP2_TYPE_MIN_CUSTOM    0xE0
#define PP2_TYPE_MAX_CUSTOM    0xEF
```

以下8个类型值的范围保留给应用程序开发人员和协议设计者进行临时实验性使用。此范围内的值永远不会被PROXY协议使用，不应用于生产功能中。

```c
#define PP2_TYPE_MIN_EXPERIMENT 0xF0
#define PP2_TYPE_MAX_EXPERIMENT 0xF7
```

以下8个类型值的范围保留用于未来使用，可能用于通过多字节类型值扩展协议。

```c
#define PP2_TYPE_MIN_FUTURE    0xF8
#define PP2_TYPE_MAX_FUTURE    0xFF
```


## 3. 实现

**HAProxy 1.5** 在两端都实现了PROXY协议的版本1：
- 当"bind"关键字中传递了"accept-proxy"设置时，监听套接字会接受该协议。此类监听器上接受的连接将表现得如同源地址确实是协议中声明的那样。这适用于日志记录、ACL、内容过滤、透明代理等场景。
- 如果在"server"行中使用了"send-proxy"设置，则可以将协议用于连接到服务器。这是基于每个服务器启用的，因此可以仅对远程服务器启用该功能，而本地服务器行为不同。如果传入连接是通过"accept-proxy"接受的，则中继的信息就是此连接的PROXY行中声明的信息。
- **HAProxy 1.5** 还实现了PROXY协议版本2作为发送方。此外，还增加了带有限制的可选SSL信息的TLV。

**Stunnel** 在版本4.45中为出站连接增加了对协议版本1的支持。

**Stud** 在2011年6月29日为出站连接增加了对协议版本1的支持。

**Postfix** 在版本2.10的`smtpd`和`postscreen`中为入站连接增加了对协议版本1的支持。

**Stud** 的补丁[5]可用于在入站连接上实现协议版本1。

**Varnish 4.1** 增加了对协议版本1和版本2的支持[6]。

**Exim** 在2014年5月13日增加了对协议版本1和版本2的支持，并将在版本4.83中发布。

**Squid** 在版本3.5中增加了对协议版本1和版本2的支持[7]。

**Jetty 9.3.0** 支持协议版本1。

**lighttpd** 在版本1.4.46中为入站连接增加了对协议版本1和版本2的支持[11]。

该协议足够简单，预计会出现其他实现，尤其是在SMTP、IMAP、FTP、RDP等环境中，客户端地址对于服务器和某些中间设备来说非常重要。实际上，一些私有部署已经在FTP和SMTP服务器上实现了该协议。

鼓励代理开发者实现此协议，因为这将使他们的产品在复杂基础设施中更加透明，并消除与日志记录和访问控制相关的许多问题。

## 4. 架构优势

### 4.1. 多层

在多层基础设施中使用PROXY协议代替透明代理具有多重优势。第一个直接的好处是可以将多个代理层串联起来，并始终呈现原始的IP地址。例如，考虑以下两层代理架构：

```
 Internet
  ,---.                     | 客户端到PX1:
 (  X  )                    | 原生协议
  `---'                     |
    |                       V
 +--+--+      +-----+
 | FW1 |------| PX1 |
 +--+--+      +-----+       | PX1到PX2: PROXY + 原生协议
    |                       V
 +--+--+      +-----+
 | FW2 |------| PX2 |
 +--+--+      +-----+       | PX2到SRV: PROXY + 原生协议
    |                       V
 +--+--+
 | SRV |
 +-----+
```

防火墙FW1接收来自互联网客户端的流量，并将其转发到反向代理PX1。
PX1添加一个PROXY头部，然后通过FW2转发到PX2。
PX2配置为读取PROXY头部并在输出中发送它，然后连接到源服务器SRV，并呈现原始客户端的地址。
由于所有TCP连接的端点都是实际的机器，并未伪造，因此返回流量通过防火墙和反向代理时不会有问题。
使用透明代理则会相当困难，因为防火墙必须处理来自DMZ中代理的客户端地址，并且必须正确地将返回流量路由到那里，而不是使用默认路由。

### 4.2. IPv4和IPv6集成

该协议还简化了IPv4和IPv6的集成：即使只有第一层（FW1和PX1）支持IPv6，也可以在整个链路仅通过IPv4连接的情况下，向目标服务器呈现原始客户端的IPv6地址。

### 4.3. 多条返回路径

使用透明代理时，无法运行多个代理，因为返回流量会遵循默认路由，而无法找到正确的代理。
有时可以通过使用多个服务器地址和策略路由来实现一些技巧，但这些方法非常有限。

使用PROXY协议，这个问题消失了，因为服务器不需要将流量路由到客户端，只需路由到转发连接的代理。
因此，即使处理多个站点，也可以在大型服务器群前运行一个代理群，并轻松工作。

这在类似云的环境中特别重要，因为在这些环境中，选择绑定到随机地址的选项很少，并且每个节点的较低处理能力通常需要多个前端节点。

以下示例说明了这种情况：虚拟化基础设施部署在三个数据中心（DC1、DC2、DC3）中。
每个数据中心使用自己的VIP，由托管提供商的三层负载均衡器处理。
该负载均衡器将流量路由到七层SSL/缓存卸载器群中，这些卸载器在其本地服务器之间进行负载均衡。
通过地理定位DNS公布VIP，以便客户端通常会固定到特定的数据中心。
由于不能保证客户端固定在一个数据中心，七层负载均衡代理需要了解其他数据中心的服务器，这些服务器可能通过托管提供商的局域网或通过互联网访问。
七层代理使用PROXY协议连接它们背后的服务器，以便即使在数据中心之间的流量也可以转发原始客户端地址，返回路径也明确无误。
使用透明代理则无法实现，因为大多数情况下，七层代理无法伪造地址，在数据中心之间这根本无法工作。

```
                           Internet

        DC1                  DC2                  DC3
       ,---.                ,---.                ,---.
      (  X  )              (  X  )              (  X  )
       `---'                `---'                `---'
         |    +-------+       |    +-------+       |    +-------+
         +----| L3 LB |       +----| L3 LB |       +----| L3 LB |
         |    +-------+       |    +-------+       |    +-------+
   ------+------- ~ ~ ~ ------+------- ~ ~ ~ ------+-------
   |||||   ||||         |||||   ||||         |||||    ||||
  50 SRV   4 PX        50 SRV   4 PX        50 SRV    4 PX
```

## 5. 安全注意事项

协议头部的版本1（可读格式）设计为可以与HTTP区分开来。它不会被解析为有效的HTTP请求，HTTP请求也不会被解析为有效的代理请求。
版本2通过使用不可解析的二进制签名来使许多产品无法正确处理该部分。
签名的设计是为了在HTTP、SSL/TLS、SMTP、FTP和POP上导致立即失败。
在LDAP和RDP服务器上也会导致中止（见第6节）。
这样可以更容易在某些连接下强制使用它，同时也确保能够快速检测到配置不正确的服务器。

实现者应非常谨慎，不能尝试自动检测是否需要解码头部，而必须仅依赖于配置参数。
实际上，如果让普通客户端有机会使用该协议，它将能够隐藏其活动或使其看起来来自其他位置。
然而，仅从一些已知来源接受头部应是安全的。

## 6. 验证

协议版本2的签名已被发送到各种协议和实现中，包括旧版本。以下是经过测试以确保在最小实现下，当提供签名时，确保最佳行为的协议和产品：

- **HTTP**：
    - Apache 1.3.33：连接中止 => 通过/最佳
    - Nginx 0.7.69：400错误请求 + 中止 => 通过/最佳
    - lighttpd 1.4.20：400错误请求 + 中止 => 通过/最佳
    - thttpd 2.20c：400错误请求 + 中止 => 通过/最佳
    - mini-httpd-1.19：400错误请求 + 中止 => 通过/最佳
    - haproxy 1.4.21：400错误请求 + 中止 => 通过/最佳
    - Squid 3：400错误请求 + 中止 => 通过/最佳

- **SSL**：
    - stud 0.3.47：连接中止 => 通过/最佳
    - stunnel 4.45：连接中止 => 通过/最佳
    - nginx 0.7.69：400错误请求 + 中止 => 通过/最佳

- **FTP**：
    - Pure-ftpd 1.0.20：3次500错误然后221再见 => 通过/最佳
    - vsftpd 2.0.1：3次530错误然后221再见 => 通过/最佳

- **SMTP**：
    - postfix 2.3：3次500错误 + 221再见 => 通过/最佳
    - exim 4.69：554错误 + 连接中止 => 通过/最佳

- **POP**：
    - dovecot 1.0.10：3次ERR + 注销 => 通过/最佳

- **IMAP**：
    - dovecot 1.0.10：5次ERR + 挂起 => 通过/非最佳

- **LDAP**：
    - openldap 2.3：中止 => 通过/最佳

- **SSH**：
    - openssh 3.9p1：中止 => 通过/最佳

- **RDP**：
    - Windows XP SP3：中止 => 通过/最佳

这意味着大多数协议和实现不会因为接收到协议签名的传入连接而感到困惑，从而在面对配置错误时避免问题。


## 7. 未来发展

协议可能会略微演变，以提供其他信息，例如传入网络接口，或者在第一个代理之前发生网络地址转换时的原始地址，但目前这并不是一个要求。
对此已经进行了深入思考，试图增加一些额外信息可能会打开“潘多拉的盒子”，包括从MAC地址到SSL客户端证书的许多信息，这将使协议变得更加复杂。
因此，当前没有这样的计划。欢迎提出改进建议。

## 8. 联系方式与链接

请使用 w@1wt.eu 向作者发送任何评论。

文档中引用了以下链接：

[1] [http://www.postfix.org/XCLIENT_README.html](http://www.postfix.org/XCLIENT_README.html)  
[2] [http://tools.ietf.org/html/rfc7239](http://tools.ietf.org/html/rfc7239)  
[3] [http://www.stunnel.org/](http://www.stunnel.org/)  
[4] [https://github.com/bumptech/stud](https://github.com/bumptech/stud)  
[5] [https://github.com/bumptech/stud/pull/81](https://github.com/bumptech/stud/pull/81)  
[6] [https://www.varnish-cache.org/docs/trunk/phk/ssl_again.html](https://www.varnish-cache.org/docs/trunk/phk/ssl_again.html)  
[7] [http://wiki.squid-cache.org/Squid-3.5](http://wiki.squid-cache.org/Squid-3.5)  
[8] [https://tools.ietf.org/html/rfc4960#appendix-B](https://tools.ietf.org/html/rfc4960#appendix-B)  
[9] [https://tools.ietf.org/rfc/rfc7301.txt](https://tools.ietf.org/rfc/rfc7301.txt)  
[10] [https://www.ietf.org/rfc/rfc3546.txt](https://www.ietf.org/rfc/rfc3546.txt)  
[11] [https://redmine.lighttpd.net/issues/2804](https://redmine.lighttpd.net/issues/2804)

### 9. 示例代码

下面的代码是接收方如何处理TCP（IPv4或IPv6）的协议头部两个版本的示例。
该函数应在读取事件时调用。由于地址按网络字节序传输，地址可以直接复制到最终的内存位置。
发送端甚至更为简单，可以从此示例代码中轻松推导出来。

```c
struct sockaddr_storage from; /* already filled by accept() */
struct sockaddr_storage to;   /* already filled by getsockname() */
const char v2sig[12] = "\x0D\x0A\x0D\x0A\x00\x0D\x0A\x51\x55\x49\x54\x0A";

/* returns 0 if needs to poll, <0 upon error or >0 if it did the job */
int read_evt(int fd)
{
  union {
      struct {
          char line[108];
      } v1;
      struct {
          uint8_t sig[12];
          uint8_t ver_cmd;
          uint8_t fam;
          uint16_t len;
          union {
              struct {  /* for TCP/UDP over IPv4, len = 12 */
                  uint32_t src_addr;
                  uint32_t dst_addr;
                  uint16_t src_port;
                  uint16_t dst_port;
              } ip4;
              struct {  /* for TCP/UDP over IPv6, len = 36 */
                   uint8_t  src_addr[16];
                   uint8_t  dst_addr[16];
                   uint16_t src_port;
                   uint16_t dst_port;
              } ip6;
              struct {  /* for AF_UNIX sockets, len = 216 */
                   uint8_t src_addr[108];
                   uint8_t dst_addr[108];
              } unx;
          } addr;
      } v2;
  } hdr;

  int size, ret;

  do {
      ret = recv(fd, &hdr, sizeof(hdr), MSG_PEEK);
  } while (ret == -1 && errno == EINTR);

  if (ret == -1)
      return (errno == EAGAIN) ? 0 : -1;

  if (ret >= 16 && memcmp(&hdr.v2, v2sig, 12) == 0 &&
      (hdr.v2.ver_cmd & 0xF0) == 0x20) {
      size = 16 + ntohs(hdr.v2.len);
      if (ret < size)
          return -1; /* truncated or too large header */

      switch (hdr.v2.ver_cmd & 0xF) {
      case 0x01: /* PROXY command */
          switch (hdr.v2.fam) {
          case 0x11:  /* TCPv4 */
              ((struct sockaddr_in *)&from)->sin_family = AF_INET;
              ((struct sockaddr_in *)&from)->sin_addr.s_addr =
                  hdr.v2.addr.ip4.src_addr;
              ((struct sockaddr_in *)&from)->sin_port =
                  hdr.v2.addr.ip4.src_port;
              ((struct sockaddr_in *)&to)->sin_family = AF_INET;
              ((struct sockaddr_in *)&to)->sin_addr.s_addr =
                  hdr.v2.addr.ip4.dst_addr;
              ((struct sockaddr_in *)&to)->sin_port =
                  hdr.v2.addr.ip4.dst_port;
              goto done;
          case 0x21:  /* TCPv6 */
              ((struct sockaddr_in6 *)&from)->sin6_family = AF_INET6;
              memcpy(&((struct sockaddr_in6 *)&from)->sin6_addr,
                  hdr.v2.addr.ip6.src_addr, 16);
              ((struct sockaddr_in6 *)&from)->sin6_port =
                  hdr.v2.addr.ip6.src_port;
              ((struct sockaddr_in6 *)&to)->sin6_family = AF_INET6;
              memcpy(&((struct sockaddr_in6 *)&to)->sin6_addr,
                  hdr.v2.addr.ip6.dst_addr, 16);
              ((struct sockaddr_in6 *)&to)->sin6_port =
                  hdr.v2.addr.ip6.dst_port;
              goto done;
          }
          /* unsupported protocol, keep local connection address */
          break;
      case 0x00: /* LOCAL command */
          /* keep local connection address for LOCAL */
          break;
      default:
          return -1; /* not a supported command */
      }
  }
  else if (ret >= 8 && memcmp(hdr.v1.line, "PROXY", 5) == 0) {
      char *end = memchr(hdr.v1.line, '\r', ret - 1);
      if (!end || end[1] != '\n')
          return -1; /* partial or invalid header */
      *end = '\0'; /* terminate the string to ease parsing */
      size = end + 2 - hdr.v1.line; /* skip header + CRLF */
      /* parse the V1 header using favorite address parsers like inet_pton.
       * return -1 upon error, or simply fall through to accept.
       */
  }
  else {
      /* Wrong protocol */
      return -1;
  }

done:
  /* we need to consume the appropriate amount of data from the socket */
  do {
      ret = recv(fd, &hdr, size, 0);
  } while (ret == -1 && errno == EINTR);
  return (ret >= 0) ? 1 : -1;
}
```
  
