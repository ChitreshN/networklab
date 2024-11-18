package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
    listener, err := net.Listen("tcp",":8000")
    if err != nil {
        fmt.Errorf("Couldnt start server", err)
    }

    fmt.Println("started server at port: 8000")

    for {
        clientConn, err := listener.Accept()
        if err != nil{
            fmt.Errorf("err: ", err)
        }
        go handleClient(clientConn)
    }
}

func handleClient(clientConn net.Conn){
    scanner := bufio.NewScanner(clientConn)

    for scanner.Scan() {
        fmt.Println(scanner.Text())
    }
}
