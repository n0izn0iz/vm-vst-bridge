package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/n0izn0iz/vm-vst-bridge-host/memconn"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	var offset int
	flag.IntVar(&offset, "offset", 0, "offset of the ring buffer in the ivshmem")
	var ringSize int
	flag.IntVar(&ringSize, "size", 16, "size of the ring buffer")
	var shmemPath string
	flag.StringVar(&shmemPath, "shmem-path", "/dev/shm/ivshmem", "path to the shared memory file")
	var client bool
	flag.BoolVar(&client, "client", false, "client mode")

	flag.Parse()

	var conn net.Conn
	if client {
		dialer := memconn.Dialer(shmemPath, ringSize, offset)
		var err error
		conn, err = dialer(context.Background(), "ivshmem")
		if err != nil {
			panic(err)
		}
		defer conn.Close()
	} else {
		lis := memconn.Listen(shmemPath, ringSize, offset)
		defer lis.Close()
		var err error
		conn, err = lis.Accept()
		defer conn.Close()
		if err != nil {
			panic(err)
		}
	}

	go func() {
		for {
			buf := make([]byte, 4096)
			_, err := conn.Read(buf)
			if err != nil {
				panic(err)
			}
			//fmt.Println("Text: ", string(buf))
			fmt.Print(string(buf))
		}
	}()

	stdinReader := bufio.NewReader(os.Stdin)
	for {
		text, _ := stdinReader.ReadString('\n')
		_, err := conn.Write([]byte(text))
		if err != nil {
			panic(err)
		}
	}
}
