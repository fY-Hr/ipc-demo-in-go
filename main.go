package main

import (
	"bufio"
	"fmt"
	"os"
	"log"
	"os/exec"
	"strings"
)

func main(){
	if len(os.Args) > 1 && os.Args[1] == "child" {
		runChildProcess()
		return
	}

	runParentProcess()
} 


func runParentProcess() {
	fmt.Println("[Parent] initializing...")

	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("[Parent] Failed to get executable path: %v", err)
	}

	cmd := exec.Command(execPath, "child")

	// defining stdin pipe
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("[Parent] Failed to get the stdin pipe: %v", err)
	}

	// defining stdout pipe
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("[Parent] Failed to get the stdout pipe: %v", err)
	}

	// starting the child process
	if err := cmd.Start(); err != nil {
		log.Fatalf("[Parent] Failed to start child process: %v", err)
	}
	
	message := "Hello from the parent process!"
	fmt.Printf("[Parent] Sending message: %s\n", message)

	// sending the message via stdinPipe
	_, err = fmt.Fprintln(stdinPipe, message)
	if err != nil {
		log.Fatalf("[Parent] Failed to write to stdin pipe: %v", err)
	}
	// close the pipe to signal the EOF (End Of File) to the child process
	stdinPipe.Close()

	// reading a response from the child process via the stdout pipe wrapped inside NewScanner
	scanner := bufio.NewScanner(stdoutPipe)
	if scanner.Scan() {
		response := scanner.Text()
		fmt.Printf("[Parent] Received response: %s\n", response)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatalf("[Parent] Child process exited with error: %v", err)
	}

	fmt.Println("[Parent] Child process ternmminated. IPC complete.")
}

func runChildProcess() {
	fmt.Fprintln(os.Stderr, "[Child] Initialized and listening...")

	// checking the message from the parent via standard input
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		// if the scanner is valid, get the Text of the scanner (the message from the parent)
		receivedMessage := scanner.Text()
		fmt.Fprintf(os.Stderr, "[Child] Message received: %s\n", receivedMessage)

		processedResponse := strings.ToUpper(receivedMessage) + " -  RECEIVED"

		// write back to the parent via standard output
		// fyi: in go, the functions within fmt, are explicitly designed to write the os.Stdout by default, so we can use it.

		// noticed in runParentProcess we are reading from stdout pipe
		fmt.Printf("[Child] Message acknowledged: %s\n", processedResponse)
		fmt.Println(processedResponse)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("[Child] Failed scanning the text from scanner")
	}

}
