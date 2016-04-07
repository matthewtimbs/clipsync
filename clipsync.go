package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/atotto/clipboard"
	"github.com/codegangsta/cli"
)

type clipContents struct {
	Contents string
	Hash     [20]byte
}

var server string
var localContents clipContents

func main() {
	server := ""
	app := cli.NewApp()
	app.Name = "clipsync"
	app.Usage = "Synchronizes clipboards between machines. Run as Server on one machine, run as client on another"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "server, s", Value: "127.0.0.1:7569", Usage: "Hostname or IP of server clipsync process and port number.  Use default for server, or specify an IP to run as client", Destination: &server},
	}
	app.Action = func(c *cli.Context) {
		runApp(server)
	}

	app.Run(os.Args)

}

func runApp(server string) {
	//endLoop := make(chan struct{})

	fmt.Printf("Press enter to quit\n")

	//Kick off the server process
	fmt.Printf("Listening on: %s\n", server)

	http.HandleFunc("/", handleServerRequest)
	fmt.Printf("...server: %s", server)
	go http.ListenAndServe(server, nil)

	//Kick off the sync process
	endLoop := pushLoop()

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

	//get local clipboard contents & update copy
	contents, _ := clipboard.ReadAll()
	hash := sha1.Sum([]byte(contents))
	localContents.Contents = contents
	localContents.Hash = hash

	//get remote
	url := "http://" + server
	fmt.Printf("host: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error contacting server: %s, error: %s", server, err.Error())
	} else {
		defer resp.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("%s", err)
		}
		fmt.Printf("%s\n", string(contents))
	}

	fmt.Printf("%x\r", hash)
}

func handleServerRequest(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "heyhey")
}
