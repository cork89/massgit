package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type BashSession struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Scanner
	stderr   *bufio.Scanner
	closeFn  func() error
	closed   bool
	closeMu  sync.Mutex
	done     chan error  // Channel to signal session completion
	ErrorLog *log.Logger // for logging errors from background processes
}

func NewBashSession() (*BashSession, error) {
	cmd := exec.Command("bash")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("StdinPipe error: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("StdoutPipe error: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("StderrPipe error: %w", err)
	}

	session := &BashSession{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   bufio.NewScanner(stdout), //buffered reading for cleaner output
		stderr:   bufio.NewScanner(stderr), //buffered reading for cleaner error handling
		done:     make(chan error, 1),
		closed:   false,
		ErrorLog: log.New(io.Discard, "BashSessionError: ", log.LstdFlags),
	}

	session.closeFn = func() error { // function to close all resources.
		session.closeMu.Lock()
		defer session.closeMu.Unlock()
		if session.closed {
			return nil
		}
		session.closed = true

		err := session.stdin.Close()
		if err != nil {
			session.ErrorLog.Printf("stdin close err: %v", err)
		}
		// sending a signal to bash helps to end faster without hanging
		if session.cmd.Process != nil {
			if err := session.cmd.Process.Signal(os.Interrupt); err != nil {
				session.ErrorLog.Printf("process signal err: %v", err)
			}
		}
		err = session.cmd.Wait()
		if err != nil {
			session.ErrorLog.Printf("cmd wait err: %v", err)
		}

		close(session.done) // signal completion even in error cases.
		return nil

	}

	err = cmd.Start()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("cmd start error: %w", err)
	}

	go session.readStdout() // Start goroutine to handle standard output.
	go session.readStderr() // Start goroutine to handle standard error.

	go func() { // Goroutine to wait for the command to finish and signal completion.
		err := cmd.Wait()
		if err != nil && !strings.Contains(err.Error(), "signal: interrupt") {
			session.ErrorLog.Printf("cmd.Wait() returned error: %v", err)
		}
		session.done <- err // Signal completion
		session.closeMu.Lock()
		session.closed = true
		session.closeMu.Unlock()

	}()

	return session, nil
}

func (s *BashSession) Execute(command string) (string, string, error) {
	if s.closed {
		return "", "", fmt.Errorf("session is closed")
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return "", "", fmt.Errorf("empty command")
	}
	command += "\n" // bash needs a newline

	_, err := io.WriteString(s.stdin, command) // Send command
	if err != nil {
		return "", "", fmt.Errorf("error writing command: %w", err) // Indicate write error
	}

	// Wait for a response from bash. This is a basic implementation. You might need to
	// implement a more sophisticated mechanism for matching commands and responses
	// if you're sending multiple commands quickly.

	// For example, sending 2 commands and immediately trying to read the stdout, can lead to reading from the 2nd commands stdout
	// In more complex cases, you might need to add a delimeter to seperate outputs.
	stdoutResult := ""
	stderrResult := ""

	// Small timeout to let the data come in. Can be increased, but should be kept small for quickness.
	// When the scanner does not have more data, break the loop.
	// Alternative would be to create a blocking channel that unblocks when data is written, but that can be complex
	for i := 0; i < 10; i++ {
		if s.stdout.Scan() {
			stdoutResult += s.stdout.Text() + "\n"
		}
		if s.stderr.Scan() {
			stderrResult += s.stderr.Text() + "\n"
		}

		// very short timeout, if nothing comes within this time, we return.
		// This is okay, since most commands will respond within this time.
		// Alternative and better approeach is to send a command, and also a seperate command
		// To print a delimeter. That way, we know when the command completes.
		// but this works as a simple implementation.
		// You can change 100ms to something else.

		time.Sleep(100 * time.Millisecond)
	}

	return strings.TrimSpace(stdoutResult), strings.TrimSpace(stderrResult), nil //Return
}

func (s *BashSession) Close() error {
	return s.closeFn()
}

func (s *BashSession) readStdout() {
	for s.stdout.Scan() {
		text := s.stdout.Text()
		// Can handle/log/process the standard output here
		// For this basic example, just logging to std output.
		fmt.Println("Bash STDOUT:", text)
	}
	if err := s.stdout.Err(); err != nil {
		s.ErrorLog.Printf("Stdout scanner error: %v", err)
	}
}

func (s *BashSession) readStderr() {
	for s.stderr.Scan() {
		text := s.stderr.Text()
		// Can handle/log/process the standard error here
		// For this basic example, just logging to std output.
		fmt.Println("Bash STDERR:", text)
	}
	if err := s.stderr.Err(); err != nil {
		s.ErrorLog.Printf("Stderr scanner error: %v", err)
	}
}

// func init() {
// 	session, err := NewBashSession()
// 	if err != nil {
// 		log.Fatalf("Failed to create bash session: %v", err)
// 	}
// 	defer session.Close()

// 	// Example usage
// 	stdout, stderr, err := session.Execute("echo hello")
// 	if err != nil {
// 		log.Printf("Error executing command: %v", err)
// 	}
// 	fmt.Println("Stdout:", stdout)
// 	fmt.Println("Stderr:", stderr)

// 	stdout, stderr, err = session.Execute("pwd")
// 	if err != nil {
// 		log.Printf("Error executing command: %v", err)
// 	}
// 	fmt.Println("Stdout:", stdout)
// 	fmt.Println("Stderr:", stderr)

// 	// Send an exit command to close the shell.
// 	session.Execute("exit") // not really needed since Close() already signals and waits for bash to end.

// 	<-session.done // Wait for session to complete
// 	fmt.Println("Bash session completed.")
// }
