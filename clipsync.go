package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/atotto/clipboard"
	"github.com/codegangsta/cli"
)

type clipContents struct {
	Contents   string
	Hash       [20]byte
	UpdateTime time.Time
}

var server string
var localContents clipContents
var serverContents clipContents

func main() {
	app := cli.NewApp()
	app.Name = "clipsync"
	app.Usage = "Synchronizes clipboards between machines. Run as Server on one machine, run as client on another"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "server, s", Value: "127.0.0.1:7569", Usage: "Hostname or IP of server clipsync process and port number.  Use default for server, or specify an IP to run as client", Destination: &server},
	}
	app.Action = func(c *cli.Context) {
		runApp()
	}

	app.Run(os.Args)

}

func runApp() {
	//fullPathServer := "http://" + server
	fmt.Printf("Press enter to quit\n")

	//Kick off the sync process
	endLoop := pushLoop()

	//Kick off the server process
	fmt.Printf("Listening on: %s\n", server)
	http.HandleFunc("/", handleServerRequest)
	go http.ListenAndServe(server, nil)

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
	resp, err := http.Get("http://" + server)
	if err != nil {
		fmt.Printf("Error contacting server: %s, error: %s\n", server, err.Error())
	} else {
		defer resp.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("%s", err)
		}
		fmt.Printf("\r%s", string(contents))
	}

	fmt.Printf("%x\r", hash)
}

func handleServerRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(serverContents)
	case "POST":
		postedClipContents := new(clipContents)
		err := json.NewDecoder(r.Body).Decode(postedClipContents)
		if err != nil {
			fmt.Printf("posted contents: %s", postedClipContents)
		} else {
			fmt.Printf(">>>postedContents %s, error: %s\n", postedClipContents, err)
		}
		serverContents = *postedClipContents
		serverContents.UpdateTime = time.Now()

	default:
		http.Error(w, "page not found", http.StatusNotFound)
	}
}
