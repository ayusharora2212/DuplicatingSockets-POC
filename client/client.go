package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sockets/utility"
	"time"

	"golang.org/x/sys/windows"
)

func main() {
	var wsaData windows.WSAData
	err := windows.WSAStartup(uint32(0x0202), &wsaData)
	if err != nil {
		fmt.Println("Wsa startup failed:", err)
	}
	defer windows.WSACleanup()
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("Error establishing connection:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connection established")
	transferToChildProcess(conn)

}

func transferToChildProcess(conn net.Conn) {
	// cmd := exec.Command("clientDup.exe")
	// if err := cmd.Start(); err != nil {
	// 	fmt.Println("Error starting child process:", err)
	// 	return
	// }
	pid, err := utility.GetProcessId("clientDup.exe")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	// pid := cmd.Process.Pid
	fmt.Println("process id:", pid)
	pipename := `\\.\pipe\socketClientDupPipe`
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
	fmt.Println("tcpConn", tcpConn)
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		fmt.Println("Error getting raw connection:", err)
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
	fmt.Println("raw handle:", sock)
	fmt.Println("WSAProtocolInfo:", info)
	// Send WSAPROTOCOL_INFO struct over the pipe
	// size := unsafe.Sizeof(info)
	var buf bytes.Buffer
	err = binary.Write(&buf, binary.LittleEndian, &info)
	if err != nil {
		fmt.Println("Error serializing the WSAProtocolInfo:", err)
	}
	data := buf.Bytes()
	var written uint32
	err = windows.WriteFile(handle, data, &written, nil)

	if err != nil {
		fmt.Println("WriteFile failed:", err)
		return
	}
	fmt.Printf("Sent %d bytes of WSAPROTOCOL_INFO to pipe\n", written)
	time.Sleep(2 * time.Second) // Wait for the child process to read the data
}
