package subcmds

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
)

var readProtoCmd = &cobra.Command{
	Use:   `read-proto --data-file=hello.bin`,
	Short: "Read proto binary file",
	Long: `
Read proto binary file.

Read proto binary file: netool read-proto -n 32
`,
	Run: func(cmd *cobra.Command, args []string) {
		filename, _ := cmd.Flags().GetString("data-file")
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Printf("read file %s error: %v\n", filename, err)
		}
		ParseProtobuf(data)
	},
}

func init() {
	readProtoCmd.Flags().StringP("data-file", "f", "", "proto binary file")
	base.AddSubCommands(readProtoCmd)
}

// Wire types
const (
	Varint          = 0
	Fixed64         = 1
	LengthDelimited = 2
	StartGroup      = 3
	EndGroup        = 4
	Fixed32         = 5
)

func ParseProtobuf(data []byte) {
	reader := bytes.NewReader(data)

	for {
		key, err := readVarint(reader)
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("Error reading key: %v\n", err)
			break
		}

		fieldNumber := key >> 3
		wireType := key & 0x7

		fmt.Printf("Field Number: %d, Wire Type: %d\n", fieldNumber, wireType)

		switch wireType {
		case Varint:
			if value, er := readVarint(reader); er != nil {
				fmt.Printf("Error reading varint value: %v\n", er)
				return
			} else {
				fmt.Printf("Varint Value: %d\n", value)
			}
		case Fixed64:
			var value uint64
			if er := binary.Read(reader, binary.LittleEndian, &value); er != nil {
				fmt.Printf("Error reading fixed64 value: %v\n", er)
				return
			}
			fmt.Printf("Fixed64 Value: %d\n", value)
		case LengthDelimited:
			length, er := readVarint(reader)
			if er != nil {
				fmt.Printf("Error reading length: %v\n", er)
				return
			}
			value := make([]byte, length)
			if _, er2 := io.ReadFull(reader, value); er2 != nil {
				fmt.Printf("Error reading length-delimited value: %v\n", er2)
				return
			}
			fmt.Printf("Length-delimited %d Value: %s\n", length, value)
		case Fixed32:
			var value uint32
			if er := binary.Read(reader, binary.LittleEndian, &value); er != nil {
				fmt.Printf("Error reading fixed32 value: %v\n", er)
				return
			}
			fmt.Printf("Fixed32 Value: %d\n", value)
		default:
			fmt.Printf("Unsupported wire type: %d\n", wireType)
			return
		}
	}
}

// binary.Uvarint(reader.Bytes())
func readVarint(reader *bytes.Reader) (uint64, error) {
	var value uint64
	var shift uint
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		value |= uint64(b&0x7F) << shift
		if b < 0x80 {
			break
		}
		shift += 7
	}
	return value, nil
}
