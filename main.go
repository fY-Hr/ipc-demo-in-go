package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		fmt.Printf("%v", os.Args)
		fmt.Printf("%v", os.Args[0])
		runChildProcess()
		return
	}

	runParentProcess()
}

func runParentProcess() {
	fmt.Println("[Parent] Initializing...")

	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("[Parent] Failed to get executable path: %v", err)
	}

	cmd := exec.Command(execPath, "child")

	// Request the kernel to create a pipe connected to the child's stdin.
	// The parent receives the write end of the pipe (io.WriteCloser),
	// while the child receives the read end as its os.Stdin.
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("[Parent] Failed to get stdin pipe: %v", err)
	}

	// Request the kernel to create another pipe connected to the child's stdout.
	// The parent receives the read end of the pipe (io.ReadCloser),
	// while the child writes to the other end through os.Stdout.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("[Parent] Failed to get stdout pipe: %v", err)
	}

	// By default, a process's stdin and stdout are connected to the terminal.
	// When using StdinPipe() and StdoutPipe(), the child's standard streams
	// are redirected to kernel-managed pipes instead.
	//
	// Parent                                         Child
	// ------                                         -----
	// stdinPipe (Write) ---> [Kernel Pipe] -------> os.Stdin
	// os.Stdout <------- [Kernel Pipe] <------ stdoutPipe (Read)
	//
	// These pipes live in kernel-managed memory (RAM), not on disk.
	//
	// Parent and child are completely separate processes.
	// They do not share variables or memory.
	// All communication happens through these kernel-managed pipes.

	// Start launches a completely new process using the same executable.
	// Parent and child are now running concurrently with separate memory.
	if err := cmd.Start(); err != nil {
		log.Fatalf("[Parent] Failed to start child process: %v", err)
	}

	message := "aaaa!"
	fmt.Printf("[Parent] Sending message: %s\n", message)

	// fmt.Fprintln converts the message into bytes, appends '\n',
	// and writes them into stdinPipe.
	//
	// Data flow:
	// Parent -> stdinPipe -> Kernel Pipe -> Child os.Stdin
	_, err = fmt.Fprintln(stdinPipe, message)
	if err != nil {
		log.Fatalf("[Parent] Failed to write to stdin pipe: %v", err)
	}

	// Closing the write end signals EOF to the child.
	// Without this, the child may continue waiting for more input.
	stdinPipe.Close()

	// stdoutPipe is an io.ReadCloser. It does not store data itself;
	// it simply reads bytes from the kernel pipe.
	//
	// Scanner wraps this reader and:
	//   - reads bytes from stdoutPipe,
	//   - keeps its own internal buffer,
	//   - splits the byte stream into lines by default.
	scanner := bufio.NewScanner(stdoutPipe)

	if scanner.Scan() {
		response := scanner.Text()
		fmt.Printf("[Parent] Received response: %s\n", response)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("[Parent] Failed reading child output: %v", err)
	}

	// Wait blocks until the child process completely exits.
	// It does NOT read data from the pipe; it simply waits for
	// the kernel to report that the child has terminated.
	if err := cmd.Wait(); err != nil {
		log.Fatalf("[Parent] Child process exited with error: %v", err)
	}

	fmt.Println("[Parent] Child process terminated. IPC complete.")
}

func runChildProcess() {
	// stderr is still connected to the terminal because we only redirected
	// stdin and stdout. This is why these messages appear directly
	// in the terminal instead of going through the parent.
	fmt.Fprintln(os.Stderr, "[Child] Initialized and listening...")

	// Fprintln is a function that allows us to write something into any io.Writer. We can put the io.Writer on the
	// first parameter of the function. and yeah, because the output streamline of the os.Stderr is still directed 
	// into the terminal, we can use it to print out the process info message.

	// fyi: F in Fprintln stand for File, but now it can be any io.Writer instead of just a file

	// In this child process, os.Stdin is no longer connected to the keyboard.
	// It is connected to the read end of the kernel pipe created by the parent.
	scanner := bufio.NewScanner(os.Stdin)

	if scanner.Scan() {
		// Scan() reads the next line from os.Stdin.
		// Text() returns the line that was read.
		receivedMessage := scanner.Text()

		fmt.Fprintf(os.Stderr, "[Child] Message received: %s\n", receivedMessage)

		processedResponse := strings.ToUpper(receivedMessage) + " - RECEIVED"

		// fmt.Println writes to os.Stdout.
		// Since the child's stdout was redirected, this output goes into
		// the kernel pipe instead of the terminal.
		//
		// Data flow:
		// Child os.Stdout -> Kernel Pipe -> stdoutPipe -> Parent Scanner
		fmt.Printf("[Child] Message acknowledged: %s\n", processedResponse)
		fmt.Println(processedResponse)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "[Child] Failed scanning stdin:", err)
	}
}
