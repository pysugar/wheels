# Wire Types

## Wire Types

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

Protobuf 定义了几种 Wire Types，每种 Wire Type 对应不同的数据编码方式。以下是基础类型与 Wire Types 的具体映射关系：

| Wire Type        | 数值    | 描述                                        | 适用的 Protobuf 类型                                                          |
|------------------|-------|-------------------------------------------|--------------------------------------------------------------------------|
| Varint           | 0     | 可变长度整数编码，适用于小整数                           | `int32`, `int64`, `uint32`, `uint64`, `sint32`, `sint64`, `bool`, `enum` |
| 64-bit           | 1     | 固定长度 64 位数据                               | `fixed64`, `sfixed64`, `double`                                          |
| Length-delimited | 2     | 长度限定的数据，如字符串、字节数组、嵌套消息等                   | `string`, `bytes`, `message`, `repeated` (packed), `map`, `oneof`, `Any` |
| 32-bit           | 5     | 固定长度 32 位数据                               | `fixed32`, `sfixed32`, `float`                                           |


### Varint（0）

**描述**：Varint 是一种可变长度的整数编码方式，适用于编码小整数。它通过使用 7 位数据和 1 位继续标志（MSB）来实现节省空间的效果。


### 64-bit（1）

**描述**：64-bit wire type 用于固定长度的 64 位数据。无论数据的实际值如何，都占用 8 字节空间。


### Length-delimited（2）

**描述**：Length-delimited wire type 用于编码长度不定的数据。首先编码数据的长度（使用 Varint），然后编码实际的数据内容。


### 32-bit（5）

**描述**：32-bit wire type 用于固定长度的 32 位数据。无论数据的实际值如何，都占用 4 字节空间。

## Protobuf 基础类型与 Wire Types 的映射

protobuf 提供了多种基础类型，这些类型根据其特性映射到不同的 wire types。以下是详细的映射关系：

| Protobuf 类型         | Go 类型                   | Wire Type             | 描述                               |
|---------------------|-------------------------|-----------------------|----------------------------------|
| `int32`             | `int32`                 | Varint (0)            | 32 位有符号整数，负数采用 Varint 编码效率低      |
| `int64`             | `int64`                 | Varint (0)            | 64 位有符号整数，负数采用 Varint 编码效率低      |
| `uint32`            | `uint32`                | Varint (0)            | 32 位无符号整数                        |
| `uint64`            | `uint64`                | Varint (0)            | 64 位无符号整数                        |
| `sint32`            | `int32`                 | Varint (0)            | 32 位有符号整数，采用 ZigZag 编码优化负数       |
| `sint64`            | `int64`                 | Varint (0)            | 64 位有符号整数，采用 ZigZag 编码优化负数       |
| `bool`              | `bool`                  | Varint (0)            | 布尔值，编码为 0 或 1                    |
| `enum`              | 自动生成的枚举类型               | Varint (0)            | 枚举类型，编码为对应的整数值                   |
| `fixed64`           | `fixed64` 或 `uint64`    | 64-bit (1)            | 固定长度 64 位整数                      |
| `sfixed64`          | `sfixed64` 或 `int64`    | 64-bit (1)            | 固定长度 64 位有符号整数                   |
| `double`            | `float64`               | 64-bit (1)            | 双精度浮点数                           |
| `string`            | `string`                | Length-delimited (2)  | UTF-8 编码的字符串                     |
| `bytes`             | `[]byte`                | Length-delimited (2)  | 任意字节序列                           |
| `embedded message`  | 指向生成的消息结构体的指针           | Length-delimited (2)  | 嵌套的消息类型                          |
| `packed repeated`   | 切片类型（如 `[]Type`）        | Length-delimited (2)  | 重复字段的打包编码                        |
| `map`               | `map[KeyType]ValueType` | Length-delimited (2)  | 键值对映射                            |
| `oneof`             | 使用接口实现的结构体              | Length-delimited (2)  | 互斥字段组                            |
| `Any`               | `*anypb.Any`            | Length-delimited (2)  | 任意类型的消息，包含类型信息和序列化数据             |
| `fixed32`           | `fixed32` 或 `uint32`    | 32-bit (5)            | 固定长度 32 位整数                      |
| `sfixed32`          | `sfixed32` 或 `int32`    | 32-bit (5)            | 固定长度 32 位有符号整数                   |
| `float`             | `float32`               | 32-bit (5)            | 单精度浮点数                           |
| `unpacked repeated` |                         |                       | 重复字段的非打包编码                       |

