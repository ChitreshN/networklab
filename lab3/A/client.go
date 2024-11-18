package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
)

const MAGIC = 0xC461
const VERSION = 1
const HELLO = 0
const DATA = 1
const ALIVE = 2
const GOODBYE = 3
const HEADER_SIZE = 16
const TIMEOUT = 5  

type Header struct {
	magic     uint16
	version   uint8
	command   uint8
	sessionId uint32
	seqNumber uint32
	clock     uint64
}

func unpackBuffer(packet []byte) (Header, string) {
    var magic uint16
    var version uint8
    var command uint8
    var sessionId uint32
    var seqNumber uint32
    var clock uint64
    message := make([]byte, len(packet)-20)

    buf := bytes.NewReader(packet)

    binary.Read(buf, binary.BigEndian, &magic)
    binary.Read(buf, binary.BigEndian, &version)
    binary.Read(buf, binary.BigEndian, &command)
    binary.Read(buf, binary.BigEndian, &seqNumber)
    binary.Read(buf, binary.BigEndian, &sessionId)
    binary.Read(buf, binary.BigEndian, &clock)
    buf.Read(message)

    header := Header{
        magic:     magic,
        version:   version,
        command:   command,
        sessionId: sessionId,
        seqNumber: seqNumber,
        clock:     clock,
    }
    return header, string(message)
}

func packBuffer(magic uint16, version uint8, command uint8, seqNumber uint32, 
                sessionId uint32, logicalClock uint64, message string) []byte{

        byteMessage := []byte(message)

        buf := new(bytes.Buffer)

        binary.Write(buf, binary.BigEndian, magic) 
        binary.Write(buf, binary.BigEndian, version) 
        binary.Write(buf, binary.BigEndian, command)  
        binary.Write(buf, binary.BigEndian, seqNumber) 
        binary.Write(buf, binary.BigEndian, sessionId) 
        binary.Write(buf, binary.BigEndian, logicalClock) 
        binary.Write(buf, binary.BigEndian, byteMessage)   

        packet := buf.Bytes()
        return packet
}

func main() {
    host := os.Args[1]
    port := os.Args[2]
    serverAddr := host+":"+port
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
    var seq uint32
    seq = 1

    sessionId := rand.Uint32()

    packet := packBuffer(MAGIC, VERSION, HELLO, 0, sessionId, 1, "")
    conn.Write(packet)

    for {
        reader := bufio.NewReader(os.Stdin)
        message, err := reader.ReadString('\n')
        message = strings.TrimSpace(message)
        command := DATA
        if err != nil || message == "q" {
            command = GOODBYE
        }
        packet := packBuffer(MAGIC,VERSION,uint8(command),seq,sessionId,1,message)
        conn.Write(packet)

        seq = seq + 1

        buffer := make([]byte, 1024)
        n, err := conn.Read(buffer)
        if err != nil {
            fmt.Println("Error reading:", err)
            return
        }

        header, _ := unpackBuffer(buffer[:n])
        switch header.command {
        case ALIVE:
            fmt.Println("alive")
        case GOODBYE:
            fmt.Println("server sent goodbye")
            os.Exit(0)
        }
    }

}
