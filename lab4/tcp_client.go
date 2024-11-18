package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main(){
    address := "localhost:8000"
    conn, err := net.Dial("tcp",address)
    if err != nil {
        fmt.Errorf("couldnt establish connection")
    }

    for {
        reader := bufio.NewReader(os.Stdin)
        message, err := reader.ReadString('\n')
        if err != nil {
            fmt.Errorf("err: ", err)
        }
        conn.Write([]byte(message))
    }
}
