# Protocol Buffers

Protocol Buffers（简称 protobuf）是由 Google 开发的一种高效的结构化数据序列化方法。它广泛应用于数据存储、通信协议等领域。
protobuf 的二进制协议设计旨在提供紧凑、高效和可扩展的数据编码方式。
本文将详细阐述 protobuf 的二进制协议，特别是其基本的 wire types，以及这些 wire types 如何映射到 protobuf 中的所有基础类型及扩展类型。

## Protocol Buffers 二进制协议概述

protobuf 的二进制协议基于“键-值”对（key-value pairs）的结构，其中每个字段由一个键（key）和一个值（value）组成。
键由字段编号（field number）和 wire type 组成。
protobuf 定义了几种 wire types，用于指示值的编码方式。

## Wire Types详解

protobuf 定义了以下几种 wire types，每种 wire type 对应不同的数据编码方式：

| Wire Type        | 数值    | 描述                              |
|------------------|-------|---------------------------------|
| Varint           | 0     | 可变长度整数编码                        |
| 64-bit           | 1     | 固定长度 64 位数据                     |
| Length-delimited | 2     | 长度限定的数据，如字符串、字节数组、嵌套消息等         |
| Start group      | 3     | （已废弃）用于嵌套消息组                    |
| End group        | 4     | （已废弃）结束嵌套消息组                    |
| 32-bit           | 5     | 固定长度 32 位数据                     |

> **注**：Wire Type 3 和 4 已在 protobuf 3 中废弃，不再推荐使用。

### Varint（0）

**描述**：Varint 是一种可变长度的整数编码方式，适用于编码小整数。它通过使用 7 位数据和 1 位继续标志（MSB）来实现节省空间的效果。

**适用类型**：

- `int32`, `int64`
- `uint32`, `uint64`
- `sint32`, `sint64`（采用 ZigZag 编码以优化负数编码）
- `bool`
- `enum`

**编码示例**：

- 整数值 1 编码为 `0x08`（字段键）
- 整数值 150 编码为 `0x96 0x01`

### 64-bit（1）

**描述**：64-bit wire type 用于固定长度的 64 位数据。无论数据的实际值如何，都占用 8 字节空间。

**适用类型**：

- `fixed64`, `sfixed64`
- `double`

**编码示例**：

- `double` 类型的值 `1.0` 编码为 `0x00 0x00 0x00 0x00 0x00 0x00 F0 3F`

### Length-delimited（2）

**描述**：Length-delimited wire type 用于编码长度不定的数据。首先编码数据的长度（使用 Varint），然后编码实际的数据内容。

**适用类型**：

- `string`
- `bytes`
- 嵌套的 `message`
- `repeated`（在 packed 模式下）
- `map`

**编码示例**：

- 字符串 `"hello"` 编码为 `0x0A 0x05 0x68 0x65 0x6C 0x6C 0x6F`
    - `0x0A`：字段键（field number = 1，wire type = 2）
    - `0x05`：长度为 5
    - `0x68 0x65 0x6C 0x6C 0x6F`：ASCII 编码的 `"hello"`

### 32-bit（5）

**描述**：32-bit wire type 用于固定长度的 32 位数据。无论数据的实际值如何，都占用 4 字节空间。

**适用类型**：

- `fixed32`, `sfixed32`
- `float`

**编码示例**：

- `float` 类型的值 `1.0` 编码为 `0x00 0x00 0x80 3F`

## Protobuf 基础类型与 Wire Types 的映射

protobuf 提供了多种基础类型，这些类型根据其特性映射到不同的 wire types。以下是详细的映射关系：

| Protobuf 类型        | Wire Type            | 描述                                        |
|--------------------|----------------------|-------------------------------------------|
| `int32`            | Varint (0)           | 32 位有符号整数，负数采用 Varint 编码效率低               |
| `int64`            | Varint (0)           | 64 位有符号整数，负数采用 Varint 编码效率低               |
| `uint32`           | Varint (0)           | 32 位无符号整数                                 |
| `uint64`           | Varint (0)           | 64 位无符号整数                                 |
| `sint32`           | Varint (0)           | 32 位有符号整数，采用 ZigZag 编码优化负数                |
| `sint64`           | Varint (0)           | 64 位有符号整数，采用 ZigZag 编码优化负数                |
| `bool`             | Varint (0)           | 布尔值，编码为 0 或 1                             |
| `enum`             | Varint (0)           | 枚举类型，编码为对应的整数值                            |
| `fixed64`          | 64-bit (1)           | 固定长度 64 位整数                               |
| `sfixed64`         | 64-bit (1)           | 固定长度 64 位有符号整数                            |
| `double`           | 64-bit (1)           | 双精度浮点数                                    |
| `string`           | Length-delimited (2) | UTF-8 编码的字符串                              |
| `bytes`            | Length-delimited (2) | 任意字节序列                                    |
| `embedded message` | Length-delimited (2) | 嵌套的消息类型                                   |
| `packed repeated`  | Length-delimited (2) | 重复字段的打包编码                                 |
| `fixed32`          | 32-bit (5)           | 固定长度 32 位整数                               |
| `sfixed32`         | 32-bit (5)           | 固定长度 32 位有符号整数                            |
| `float`            | 32-bit (5)           | 单精度浮点数                                    |

> **注**：在 protobuf3 中，`groups` 类型已被废弃，故不再推荐使用 wire types 3 和 4。

### 有符号与无符号整数的优化

对于有符号整数类型，如 `int32` 和 `int64`，在负数值较多时，Varint 编码效率较低，因为 Varint 对于负数需要占用更多的字节。为了解决这一问题，protobuf 引入了 `sint32` 和 `sint64` 类型，采用 ZigZag 编码方式，将有符号整数映射为无符号整数，从而提高负数编码的效率。

**ZigZag 编码**：

- 将负数映射为奇数，无符号数的偶数表示正数。
- 公式：`zigzag(n) = (n << 1) ^ (n >> 31)`（32 位）
- 示例：
    - `0` → `0`
    - `-1` → `1`
    - `1` → `2`
    - `-2` → `3`
    - `2` → `4`

### Repeated 与 Map 类型

**Repeated 类型**：

- 非打包模式：每个元素单独编码，使用相同的字段编号。
- 打包模式（packed）：所有元素作为一个长度限定的字节序列编码，适用于数值类型以减少开销。

**Map 类型**：

- 作为嵌套的消息类型实现，每个键值对作为一个独立的嵌套消息编码。

## 扩展类型的处理

protobuf 支持通过扩展（extensions）来添加额外的字段，但在 protobuf3 中，推荐使用 `Any` 类型或自定义的嵌套消息来实现扩展功能。

### `Any` 类型

`Any` 类型允许在消息中嵌入任意类型的消息。它通过存储消息的类型 URL 和序列化后的字节数据来实现。

**示例**：

```protobuf
import "google/protobuf/any.proto";

message Wrapper {
  google.protobuf.Any payload = 1;
}
```

### 自定义嵌套消息

通过定义嵌套的消息类型，可以实现灵活的扩展。

**示例**：

```protobuf
message BaseMessage {
  int32 id = 1;
  string name = 2;
}

message ExtendedMessage {
  BaseMessage base = 1;
  string extra_info = 2;
}
```

## 总结

protobuf 的二进制协议通过定义不同的 wire types（Varint、64-bit、Length-delimited、32-bit）来高效地编码各种基础类型及扩展类型。
理解这些 wire types 及其与 protobuf 类型的映射关系，有助于更好地设计和优化 protobuf 消息结构，以实现高效的数据序列化和反序列化。

## 附录

您提出的问题涉及 Protocol Buffers（protobuf）的基础类型及其与 Wire Types 的映射关系，以及 `Any` 类型的处理方式，以及 ZigZag 和 Varint 编码的详细规范。以下是详细的解答：

---

## **1. Protobuf 的基础类型及其与 Wire Types 的映射关系**

Protobuf 提供了多种基础类型，这些类型根据其特性映射到不同的 Wire Types，以实现高效的二进制编码。以下是 Protobuf 的所有基础类型及其对应的 Wire Types：

### **1.1. 基础类型列表**

| Protobuf 类型       | 描述                                   |
|---------------------|----------------------------------------|
| `double`            | 双精度浮点数，64 位                     |
| `float`             | 单精度浮点数，32 位                     |
| `int32`             | 32 位有符号整数                         |
| `int64`             | 64 位有符号整数                         |
| `uint32`            | 32 位无符号整数                         |
| `uint64`            | 64 位无符号整数                         |
| `sint32`            | 32 位有符号整数，使用 ZigZag 编码         |
| `sint64`            | 64 位有符号整数，使用 ZigZag 编码         |
| `fixed32`           | 固定长度 32 位无符号整数                 |
| `fixed64`           | 固定长度 64 位无符号整数                 |
| `sfixed32`          | 固定长度 32 位有符号整数                 |
| `sfixed64`          | 固定长度 64 位有符号整数                 |
| `bool`              | 布尔值，`true` 或 `false`               |
| `string`            | UTF-8 编码的字符串                       |
| `bytes`             | 任意字节序列                             |
| `enum`              | 枚举类型                                 |
| `message`           | 嵌套的消息类型                           |
| `repeated`          | 重复字段，可用于列表或数组               |
| `map`               | 键值对映射                               |
| `oneof`             | 一组互斥字段                             |
| `Any`               | 任意类型的消息，包含类型信息和序列化数据   |

### **1.2. Wire Types 与基础类型的映射关系**

Protobuf 定义了几种 Wire Types，每种 Wire Type 对应不同的数据编码方式。以下是基础类型与 Wire Types 的具体映射关系：

| Wire Type | 数值 | 描述                                     | 适用的 Protobuf 类型                             |
|-----------|------|------------------------------------------|--------------------------------------------------|
| Varint    | 0    | 可变长度整数编码，适用于小整数           | `int32`, `int64`, `uint32`, `uint64`, `sint32`, `sint64`, `bool`, `enum` |
| 64-bit    | 1    | 固定长度 64 位数据                       | `fixed64`, `sfixed64`, `double`                  |
| Length-delimited | 2 | 长度限定的数据，如字符串、字节数组、嵌套消息等 | `string`, `bytes`, `message`, `repeated` (packed), `map`, `oneof`, `Any` |
| 32-bit    | 5    | 固定长度 32 位数据                       | `fixed32`, `sfixed32`, `float`                   |

**注**：Wire Type 3 和 4 已在 protobuf 3 中废弃，不再推荐使用。

### **1.3. 表格总结**

| Protobuf 类型 | Wire Type | 描述                                     |
|---------------|-----------|------------------------------------------|
| `double`      | 1         | 64-bit，双精度浮点数                     |
| `float`       | 5         | 32-bit，单精度浮点数                     |
| `int32`       | 0         | Varint，32 位有符号整数                   |
| `int64`       | 0         | Varint，64 位有符号整数                   |
| `uint32`      | 0         | Varint，32 位无符号整数                   |
| `uint64`      | 0         | Varint，64 位无符号整数                   |
| `sint32`      | 0         | Varint，32 位有符号整数，ZigZag 编码        |
| `sint64`      | 0         | Varint，64 位有符号整数，ZigZag 编码        |
| `fixed32`     | 5         | 32-bit，固定长度无符号整数                 |
| `fixed64`     | 1         | 64-bit，固定长度无符号整数                 |
| `sfixed32`    | 5         | 32-bit，固定长度有符号整数                 |
| `sfixed64`    | 1         | 64-bit，固定长度有符号整数                 |
| `bool`        | 0         | Varint，布尔值                             |
| `string`      | 2         | Length-delimited，UTF-8 字符串              |
| `bytes`       | 2         | Length-delimited，字节序列                    |
| `enum`        | 0         | Varint，枚举类型                             |
| `message`     | 2         | Length-delimited，嵌套消息类型                |
| `repeated`    | 0 或 2    | Varint（非打包模式）或 Length-delimited（打包模式） |
| `map`         | 2         | Length-delimited，键值对映射                    |
| `oneof`       | 2         | Length-delimited，互斥字段组                     |
| `Any`         | 2         | Length-delimited，包含类型信息和序列化数据          |

---

## **2. `Any` 类型的 WireType 映射及在 Go 中的表示**

### **2.1. `Any` 类型的 WireType 映射**

`Any` 类型在 Protobuf 中用于封装任意类型的消息。它通过存储被封装消息的类型 URL 和序列化后的字节数据来实现这一功能。

- **Wire Type**：`Any` 类型映射到 `Length-delimited`（Wire Type 2），因为它包含了一个长度限定的数据块。

### **2.2. `Any` 类型在 Go 中的表示**

在 Go 中，Protobuf 的 `Any` 类型通常映射为 `*anypb.Any`，其中 `anypb` 是 Protobuf 官方提供的包，用于处理 `Any` 类型。

- **Go 类型**：`*anypb.Any`

  ```go
  import "google.golang.org/protobuf/types/known/anypb"

  var a anypb.Any
  ```

- **说明**：虽然在一些自定义实现中可能会使用 `interface{}` 来表示 `Any` 类型，但官方推荐使用 `*anypb.Any` 类型，因为它提供了内置的方法来封装和解封装任意 Protobuf 消息，确保类型安全和正确的序列化/反序列化。

---

## **3. ZigZag 编码的详细规范**

### **3.1. ZigZag 编码简介**


---

## **4. Varint 编码的规范**

### **4.1. Varint 编码简介**

Varint（Variable-length integer）是一种用于编码整数的可变长度的编码方式，旨在减少存储空间，特别是对于小整数。Varint 编码通过使用每个字节的最高位（MSB）作为续写标志位来实现。

### **4.2. Varint 的工作原理**

- **结构**：每个字节的前 7 位用于存储数据，最高位（第 8 位）作为续写标志。
    - **最高位为 1**：表示后续还有字节。
    - **最高位为 0**：表示这是最后一个字节。

- **编码步骤**：
    1. **分割整数**：将待编码的整数按 7 位一组，从最低位开始分割。
    2. **设置续写标志**：
        - 对于除最后一组之外的每一组，设置最高位为 1。
        - 对于最后一组，设置最高位为 0。
    3. **按顺序输出字节**：从最低有效字节到最高有效字节顺序输出。

### **4.3. 编码示例**

| 整数值 | 二进制表示   | 分组（7 位） | 每组字节（含续写标志） | Varint 编码结果（十六进制） |
|--------|--------------|--------------|------------------------|----------------------------|
| 1      | `0000 0001`  | `0000001`    | `00000001`              | `0x01`                     |
| 150    | `1001 0110`  | `0010110`, `0000001` | `10010110`（续写），`00000001`（结束） | `0x96 0x01`               |
| 300    | `1 0010 1100`| `001100`, `0000010` | `10101100`（续写），`00000010`（结束） | `0xAC 0x02`               |

**详细步骤**：

以整数 300 为例：

1. **二进制表示**：`1 0010 1100`（9 位）
2. **分组**：
    - 第一组（低 7 位）：`0010 1100`（44）
    - 第二组（高位）：`0000 0010`（2）
3. **设置续写标志**：
    - 第一组：`1 0010 1100` → `1010 1100`（0xAC）
    - 第二组：`0 0000 0010` → `0000 0010`（0x02）
4. **编码结果**：`0xAC 0x02`

### **4.4. Varint 的解码过程**

1. **初始化**：设定一个结果变量 `result = 0`，以及一个位移量 `shift = 0`。
2. **逐字节读取**：
    - 读取一个字节。
    - 提取前 7 位，并将其左移 `shift` 位后加到 `result`。
    - 如果最高位为 0，结束解码。
    - 如果最高位为 1，继续读取下一个字节，并增加 `shift`（通常每次增加 7）。
3. **完成**：解码后的 `result` 即为原始整数。

**示例**：解码 `0x96 0x01`

1. **第一个字节**：`0x96` → `1001 0110`
    - 数据位：`001 0110`（22）
    - 续写标志：1
    - `result += 22 << 0` → `result = 22`
    - 增加 `shift`：`shift = 7`
2. **第二个字节**：`0x01` → `0000 0001`
    - 数据位：`000 0001`（1）
    - 续写标志：0
    - `result += 1 << 7` → `result = 22 + 128 = 150`
    - 解码结束
3. **结果**：150

### **4.5. Varint 编码的优势与局限**

**优势**：

- **节省空间**：对于小整数，Varint 编码非常高效，使用较少的字节表示。
- **灵活性**：可以编码任意大小的整数，只受限于实现的最大字节数。

**局限**：

- **大整数开销大**：对于非常大的整数，Varint 编码可能会使用较多字节。
- **非对齐访问**：由于编码长度不固定，随机访问特定位置的数据较为困难。

---

## **总结**

Protocol Buffers 的二进制协议通过不同的 Wire Types 高效地编码各种基础类型及扩展类型。理解这些基础类型与 Wire Types 的映射关系，有助于更好地设计和优化 Protobuf 消息结构。此外，深入了解 ZigZag 和 Varint 编码规范，可以进一步优化数据的序列化和反序列化过程，提升系统的性能和效率。

如果您有更多关于 Protobuf 的问题或需要进一步的详细解释，请随时提问！