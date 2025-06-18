package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sockets/socketDup/azcri"
	"sockets/utility"
	"time"

	"golang.org/x/sys/windows"
	"google.golang.org/protobuf/proto"
)

func main() {
	var wsaData windows.WSAData
	err := windows.WSAStartup(uint32(0x0202), &wsaData)
	if err != nil {
		fmt.Println("Wsa startup failed:", err)
	}
	fmt.Println("startup success")
	defer windows.WSACleanup()
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
	} else {
		fmt.Println("Current process executable:", filepath.Base(exePath))
	}
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Println("Error listening to port", err)
		return
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Error establishing connection:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connection established")
	transferToChildProcess(conn)

}

func transferToChildProcess(conn net.Conn) {
	// cmd := exec.Command("./serverDup.exe")

	// if err := cmd.Start(); err != nil {
	// 	fmt.Println("Error starting child process:", err)
	// 	return
	// }
	// pid := cmd.Process.Pid
	pid, err := utility.GetProcessId("serverDup.exe")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("process id:", pid)
	time.Sleep(2 * time.Second) // wait for the child process to start
	pipename := `\\.\pipe\socketServerDupPipe`
	pipenamePtr := windows.StringToUTF16Ptr(pipename)
	handle, err := windows.CreateFile(
		pipenamePtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,   // no sharing
		nil, // default security attributes
		windows.OPEN_EXISTING,
		0, // no special attributes
		0, // no template file
	)
	if err != nil {
		fmt.Println("Error in connecting to named pipe:", err)
		return
	}
	defer windows.CloseHandle(handle)
	fmt.Println("connected to named pipe successfully")
	tcpConn := conn.(*net.TCPConn)
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		fmt.Println("Error getting raw conenction:", err)
		return
	}
	var sock windows.Handle
	err = rawConn.Control(func(fd uintptr) {
		sock = windows.Handle(fd)
	})
	if err != nil {
		fmt.Println("Error getting socket handle:", err)
		return
	}
	var info windows.WSAProtocolInfo
	err = windows.WSADuplicateSocket(sock, uint32(pid), &info)
	if err != nil {
		fmt.Println("Error duplicating socket:", err)
		return
	}

	fmt.Println("information about socket:", info)
	fmt.Println("raw handle:", sock)
	var buf bytes.Buffer
	err = binary.Write(&buf, binary.LittleEndian, &info)
	if err != nil {
		fmt.Println("Error in serializing WSAProtocolInfo:", err)
		return
	}
	data := buf.Bytes()
	protobufData := &azcri.WSADuplicateSocketInfo{
		SocketId:     int32(sock),
		ProtocolInfo: data,
	}
	serializedDta, err := proto.Marshal(protobufData)
	if err != nil {
		fmt.Println("Error serializing protobuf data:", err)
		return
	}
	fmt.Println(serializedDta)
	var written uint32
	err = windows.WriteFile(handle, serializedDta, &written, nil)
	if err != nil {
		fmt.Println("Error writing to named pipe:", err)
		return
	}
	fmt.Printf("Sent %d bytes of WSAPROTOCOL_INFO to pipe\n", written)
	time.Sleep(2 * time.Second)

}
