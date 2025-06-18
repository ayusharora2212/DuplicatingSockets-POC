package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func main() {
	pipename := `\\.\pipe\socketClientDupPipe`
	pipenamePtr := windows.StringToUTF16Ptr(pipename)
	handle, err := windows.CreateNamedPipe(
		pipenamePtr,
		windows.PIPE_ACCESS_DUPLEX,
		windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT,
		windows.PIPE_UNLIMITED_INSTANCES,
		1024*16, // output buffer size
		1024*16, // input buffer size
		0,       // default timeout
		nil,     // default security attributes
	)
	if err != nil {
		fmt.Println("Error in creatind named pipe:", err)
		return
	}
	defer windows.CloseHandle(handle)
	fmt.Println("Named pipe created successfully:")
	err = windows.ConnectNamedPipe(handle, nil)
	if err != nil {
		fmt.Println("Error in connecting with named pipe:", err)
		return
	}
	fmt.Println("successfully connected to named pipe")
	buf := make([]byte, binary.Size(windows.WSAProtocolInfo{}))
	var info windows.WSAProtocolInfo
	var bytesRead uint32
	err = windows.ReadFile(handle, buf, &bytesRead, nil)
	if err != nil {
		fmt.Println("Error reading from named pipe:", err)
		return
	}
	reader := bytes.NewReader(buf)
	err = binary.Read(reader, binary.LittleEndian, &info)
	if err != nil {
		fmt.Println("Error decoding WSAProtocolInfo:", err)
		return
	}
	fmt.Printf("Read %d bytes from named pipe\n", bytesRead)
	fmt.Println("WSAProtocolInfo:", info)
	var wsaData windows.WSAData
	err = windows.WSAStartup(uint32(0x0202), &wsaData)
	if err != nil {
		fmt.Println("WSAStartup failed", err)
		return
	}
	newSock, err := windows.WSASocket(info.AddressFamily, info.SocketType, info.Protocol, &info, 0, 0)
	if err != nil {
		fmt.Println("new socket creation failed:", err)
		return
	}
	fmt.Println("new socket created successfully with handle:", newSock)
	handleConnection(newSock)
}

func handleConnection(sock windows.Handle) {
	defer windows.Closesocket(sock)
	// Data to send
	go func() {
		data := make([]byte, 4096)
		for {
			var wsabuf windows.WSABuf
			wsabuf.Len = uint32(len(data))
			wsabuf.Buf = (*byte)(unsafe.Pointer(&data[0]))

			var bytesRead uint32
			var flags uint32 = 0

			err := windows.WSARecv(sock, &wsabuf, 1, &bytesRead, &flags, nil, nil)
			if err != nil {
				fmt.Println("Error receiving data from socket:", err)
				return
			}
			if bytesRead > 0 {
				received := strings.TrimSpace(string(data[:bytesRead]))
				fmt.Println("Received from server:", received)
				if received == "exit" {
					fmt.Println("Server requested exit.")
					os.Exit(0)
				}
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Enter message to send: ")
		if !scanner.Scan() {
			break
		}
		text := scanner.Text()
		message := []byte(text)
		var wsabuf windows.WSABuf
		wsabuf.Buf = (*byte)(unsafe.Pointer(&message[0]))
		wsabuf.Len = uint32(len(message))
		var bytesSent uint32

		err := windows.WSASend(sock, &wsabuf, 1, &bytesSent, 0, nil, nil)
		if err != nil {
			fmt.Println("Error sending data to socket:", err)
			return
		}

		if text == "exit" {
			fmt.Println("Exiting...")
			break
		}
	}
}
