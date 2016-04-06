package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/atotto/clipboard"
	"github.com/codegangsta/cli"
)

type clipContents struct {
	contents string
	hash     [20]byte
}

func main() {
	server, port := "", ""
	app := cli.NewApp()
	app.Name = "clipsync"
	app.Usage = "Synchronizes clipboards between machines. Run as Server on one machine, run as client on another"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "server, s", Value: "127.0.0.1", Usage: "Hostname or IP of server clipsync process.  Use default for server, or specify an IP to run as client", Destination: &server},
		cli.StringFlag{Name: "port, p", Value: "7564", Usage: "Port number", Destination: &port},
	}
	app.Action = func(c *cli.Context) {
		runApp(server, int(port))
	}

	app.Run(os.Args)

}

func runApp(server string, port int) {
	//endLoop := make(chan struct{})

	fmt.Printf("Press enter to quit")

	//Kick off the sync process
	endLoop := pushLoop()

	//Kick off the server process
	http.HandleFunc("/", handleServerRequest)
	//fmt.Println(c.Args()[])
	//http.ListenAndServe(c.Args()["port, p"])

	//Block until done.
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
	fmt.Printf("quitting!!!")
	endLoop <- struct{}{} //send literal of type struct{}
}

func pushLoop() chan struct{} {
	ticker := time.NewTicker(500 * time.Millisecond)
	quit := make(chan struct{})

	//print something here mjt

	go func() {
		for {
			select {
			case <-ticker.C:
				//do stuff
				syncClipboard()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return quit
}

func syncClipboard() {

	//get local clipboard contents
	contents, _ := clipboard.ReadAll()
	hash := sha1.Sum([]byte(contents))

	fmt.Printf("%x\n", hash)
}

func handleServerRequest(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "heyhey")
}
