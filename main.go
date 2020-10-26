package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"
)

// fixme: add ids to prevent misuse

func main() {
	var offset int
	flag.IntVar(&offset, "offset", 0, "offset of the ring buffer in the ivshmem")
	var ringSize int
	flag.IntVar(&ringSize, "size", 16, "size of the ring buffer")
	var shmemPath string
	flag.StringVar(&shmemPath, "shmem-path", "/dev/shm/ivshmem", "path to the shared memory file")
	var client bool
	flag.BoolVar(&invert, "client", false, "client mode")

	flag.Parse()

	if client {

	} else {
		lis := Listen(shmemPath, ringSize, offset)
	}

	var readerOffset C.size_t
	var writerOffset C.size_t
	var readerIndex int
	var writerIndex int
	if !invert {
		readerOffset = C.size_t(offset)
		writerOffset = C.size_t(offset + (ringSize / 2))
		readerIndex = 0
		writerIndex = 1
	} else {
		readerOffset = C.size_t(offset + (ringSize / 2))
		writerOffset = C.size_t(offset)
		readerIndex = 1
		writerIndex = 0
	}

	go func() {
		//defer func() { fmt.Println("Reader died!") }()

		mem.readIndices[readerIndex] = mem.writeIndices[readerIndex]
		//fmt.Println("readIndex: ", mem.readIndices[readerIndex])
		for {
			//fmt.Println("_______________________")
			var readUntil C.uint64_t
			for {
				readUntil = mem.writeIndices[readerIndex]
				if mem.readIndices[readerIndex] != readUntil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			var buf []byte
			var toRead C.uint64_t
			if mem.readIndices[readerIndex] < readUntil {
				toRead = readUntil - mem.readIndices[readerIndex]
			} else {
				toRead = readUntil + (size - mem.readIndices[readerIndex])
			}
			//fmt.Println("toRead: ", toRead)
			for i := C.uint64_t(0); i < toRead; i++ {
				b := C.readByte(mem, readerOffset+((mem.readIndices[readerIndex]+i)%size))
				buf = append(buf, byte(b))
			}
			//fmt.Println("__")
			//fmt.Println("Text: ", string(buf))
			fmt.Print(string(buf))
			mem.readIndices[readerIndex] = readUntil
			//fmt.Println("readIndex: ", mem.readIndices[readerIndex])
		}
	}()

	stdinReader := bufio.NewReader(os.Stdin)
	//fmt.Println("writeIndex: ", mem.writeIndices[writerIndex])
	for {
		//fmt.Println("_______________________")
		//fmt.Print("Enter text: ")
		text, _ := stdinReader.ReadString('\n')

		toWrite := C.uint64_t(len(text))
		//fmt.Println("toWrite: ", toWrite)
		baseIndex := mem.writeIndices[writerIndex]
		for i := C.uint64_t(0); i < toWrite; i++ {
			for {
				writeUntil := mem.readIndices[writerIndex]
				var canWrite C.uint64_t
				if writeUntil == mem.writeIndices[writerIndex] {
					canWrite = size - 1
				} else if writeUntil < mem.writeIndices[writerIndex] {
					canWrite = (size - mem.writeIndices[writerIndex]) + writeUntil - 1
				} else {
					canWrite = writeUntil - mem.writeIndices[writerIndex] - 1
				}
				if canWrite > 0 {
					break
				}
				//fmt.Println("Waiting for read..")
				time.Sleep(10 * time.Millisecond)
			}
			C.writeByte(mem, writerOffset+((baseIndex+i)%size), C.char(text[i]))
			mem.writeIndices[writerIndex] = (mem.writeIndices[writerIndex] + 1) % size
		}
		//fmt.Println("writeIndex: ", mem.writeIndices[writerIndex])
	}
}
