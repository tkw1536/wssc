package main

//go:generate gogenlicense -m

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

func main() {
	if legalFlag {
		fmt.Println(Notices)
		return
	}

	if len(args) != 1 {
		log.Fatalf("Usage: wssc address")
	}

	conn, _, err := websocket.DefaultDialer.DialContext(globalContext, args[0], nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("Established connection to %s", conn.RemoteAddr())

	go writeMessages(conn)

	readMessages(conn)
	log.Println("Closing connection")
}

func readMessages(conn *websocket.Conn) {
	for globalContext.Err() == nil {
		_, content, err := conn.ReadMessage()
		if ce, ok := err.(*websocket.CloseError); ok {
			log.Printf("Websocket closed connection: %s", ce.Text)
			return
		}
		if err != nil {
			log.Fatalf("Error receiving message: %s", err)
		}
		os.Stdout.Write(content)
	}
}

func writeMessages(conn *websocket.Conn) {
	scanner := bufio.NewScanner(os.Stdin)
	for globalContext.Err() == nil {
		text, err := readText(scanner, globalContext)
		if err != nil {
			log.Fatalf("Error reading message from stdin: %s", err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
			log.Fatalf("Error writing message: %s", err)
		}
	}
}

func readText(scanner *bufio.Scanner, ctx context.Context) (string, error) {
	textChan := make(chan string)
	errChan := make(chan error)
	go func() {
		if !scanner.Scan() {
			errChan <- scanner.Err()
			return
		}
		textChan <- scanner.Text()
	}()

	select {
	case text := <-textChan:
		return text, nil
	case err := <-errChan:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

var globalContext context.Context

func init() {
	var cancel context.CancelFunc
	globalContext, cancel = context.WithCancel(context.Background())

	cancelChan := make(chan os.Signal)
	signal.Notify(cancelChan, os.Interrupt)

	go func() {
		<-cancelChan
		cancel()
	}()
}

var legalFlag bool
var args []string

func init() {
	flag.BoolVar(&legalFlag, "legal", legalFlag, "Display legal notices and exit")

	flag.Parse()
	args = flag.Args()
}
