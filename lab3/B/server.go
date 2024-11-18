package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
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

type SessionData struct {
	seqNumber uint32
	clock     uint64
    addr      net.Addr
}

var sessionMap = struct {
	sync.RWMutex
	sessions map[uint32]SessionData
}{sessions: make(map[uint32]SessionData)}

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



func updateSession(header Header,addr net.Addr) {
	sessionMap.Lock()
	defer sessionMap.Unlock()

	if session, exists := sessionMap.sessions[header.sessionId]; exists {
		session.seqNumber = header.seqNumber
		session.clock = header.clock
		sessionMap.sessions[header.sessionId] = session
	} else {
		sessionMap.sessions[header.sessionId] = SessionData{
			seqNumber: header.seqNumber,
			clock:     header.clock,
            addr:      addr,
		}
	}

}

func serverRoutine(conn net.PacketConn) {

    buffer := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			fmt.Println("Error reading:", err)
			continue
		}

		packet := buffer[:n]

		header, message := unpackBuffer(packet)

		if header.version != 1 || header.magic != 0xC461 {
			continue
		}


        switch header.command {
            case HELLO:
                if header.seqNumber == 0 {
                    fmt.Printf("%d[%d]: %s\n", header.sessionId,header.seqNumber,"session created")
                    continue
                }

            case DATA:
                curSeq := sessionMap.sessions[header.sessionId].seqNumber
                if  curSeq == header.seqNumber {
                    fmt.Println("duplicate packet")
                }
                if curSeq + 1 < header.seqNumber {
                    fmt.Println(curSeq, header.seqNumber)
                    fmt.Println("lost packet!")
                }
                if curSeq > header.seqNumber {
                    reply := packBuffer(MAGIC, VERSION, GOODBYE, 
                    header.seqNumber, header.sessionId, header.clock, "")
                    _, err = conn.WriteTo(reply, addr)
                    if err != nil {
                        fmt.Println("Error sending:", err)
                    }
                    delete(sessionMap.sessions,header.sessionId)
                    continue
                }
                fmt.Printf("%d[%d]: %s\n", header.sessionId,header.seqNumber,message)
                updateSession(header,addr)

            case GOODBYE:
                fmt.Println("client sent goodbye")
                reply := packBuffer(MAGIC, VERSION, GOODBYE, 
                header.seqNumber, header.sessionId, header.clock, "")
                _, err = conn.WriteTo(reply, addr)
                if err != nil {
                    fmt.Println("Error sending:", err)
                }
                delete(sessionMap.sessions,header.sessionId)
                continue
        }

        reply := packBuffer(MAGIC, VERSION, ALIVE, header.seqNumber, 
        header.sessionId, header.clock, "")

		_, err = conn.WriteTo(reply, addr)
		if err != nil {
			fmt.Println("Error sending:", err)
		}
	}
}

func input(conn net.PacketConn){
    for{
        reader := bufio.NewReader(os.Stdin)
        input,_ := reader.ReadString('\n')
        input = strings.TrimSpace(input)
        if input == "q" {
            for k := range sessionMap.sessions {
                addr := sessionMap.sessions[k].addr
                reply := packBuffer(MAGIC,VERSION,GOODBYE,
                sessionMap.sessions[k].seqNumber,1,1,"")
                conn.WriteTo(reply,addr)
            }
        }
        os.Exit(0)
    }
}

func main() {
    port := os.Args[1]
	conn, err := net.ListenPacket("udp", ":"+port)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
    defer conn.Close()
    go serverRoutine(conn)
    go input(conn)

    select{}
}

