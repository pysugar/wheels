package subcmds_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/pysugar/wheels/cmd/subcmds"
	"github.com/pysugar/wheels/examples/user"
	"github.com/pysugar/wheels/serial"
)

const (
	testUserFile = "/tmp/user.pb.bin"
	testAcctFile = "/tmp/acct.pb.bin"
)

func TestProtoWriteToFile(t *testing.T) {
	acct := &user.Account{Username: "gosuger", Password: "xxxxxx"}
	u := &user.User{
		Level:   127,
		Email:   "hello@world.com",
		Account: serial.Encode(acct),
	}

	data, err := proto.Marshal(u)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(testUserFile)
	if err != nil {
		log.Fatalf("Failed to create file: %v\n", err)
	}
	defer file.Close()

	n, err := file.Write(data)
	if err != nil {
		log.Fatalf("Failed to write data to file: %v\n", err)
	}

	log.Printf("Proto object successfully written to port_list.bin, length: %d", n)

	acctData, err := proto.Marshal(acct)
	if err != nil {
		panic(err)
	}

	acctFile, err := os.Create(testAcctFile)
	if err != nil {
		log.Fatalf("Failed to create file: %v\n", err)
	}
	defer file.Close()

	n, err = acctFile.Write(acctData)
	if err != nil {
		log.Fatalf("Failed to write data to file: %v\n", err)
	}

	log.Printf("Proto object successfully written to port_list.bin, length: %d", n)
}

func TestProtoReadFromFile(t *testing.T) {
	data, err := os.ReadFile(testUserFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v\n", err)
	}

	u := &user.User{}
	if er := proto.Unmarshal(data, u); er != nil {
		t.Fatalf("unmashal error: %v\n", er)
	}

	t.Logf("user: %v\n", u)
	acct, err := serial.Decode(u.Account)
	if err != nil {
		t.Fatalf("serial decode error: %v\n", err)
	}
	t.Logf("acct: %v\n", acct)
}

func TestParseProtobuf(t *testing.T) {
	data, err := os.ReadFile(testUserFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v\n", err)
	}

	fmt.Printf("%v\n", data)
	subcmds.ParseProtobuf(data)

	fmt.Println("------------")
	data2, err := os.ReadFile(testAcctFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v\n", err)
	}
	fmt.Printf("%v\n", data2)
	subcmds.ParseProtobuf(data2)
}
