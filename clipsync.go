package main

//TODO Extract server and client, common api.
//TODO refactor get/post
//TODO Extract ClipContents with JSON methods
//TODO rename SErver to URL

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
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

	fmt.Printf("Syncing clipboard with server %s", server)

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

	//TODO MJT add error handling

	//get OS clipboard contents
	contents, _ := clipboard.ReadAll()
	hash := sha1.Sum([]byte(contents))

	//Update clipsync client's copy if it's empty or the OS clipboard copy has changed
	if localContents.Hash != hash {
		localContents.Contents = contents
		localContents.Hash = hash
		localContents.UpdateTime = time.Now() //Note: won't work if machines are in different timezones
		fmt.Printf("Debug: Client Updated Local Copy From OS\n")
	}

	remoteServerClipContents := new(clipContents)

	//Get clipsync server's copy
	resp, err := http.Get("http://" + server)

	if err != nil {
		fmt.Printf("Error contacting server: %s, error: %s\n", server, err.Error())
	} else {
		err := json.NewDecoder(resp.Body).Decode(remoteServerClipContents)
		if err != nil {
			fmt.Printf("error: %s\n", err)
		}
		fmt.Printf(">>>>>>>>>>>%s", remoteServerClipContents) //TODO MJT, not getting good stuff here.
	}

	fmt.Printf("local: %s, server %s\n", localContents.Hash, remoteServerClipContents.Hash)

	//if clipsync client copy != clipsync server copy...
	if localContents.Hash != remoteServerClipContents.Hash {
		if remoteServerClipContents == nil || localContents.UpdateTime.Sub(remoteServerClipContents.UpdateTime) > 1 {
			//if server has nothing or clipsync client copy newer, post to server
			byteArray, _ := json.Marshal(localContents)
			r := bytes.NewReader(byteArray)
			if err == nil {
				http.NewRequest("POST", server, r)
				fmt.Printf("Debug: Client Pushed Clipboard to Server\n")
			}

		} else {
			//clipsync server copy newer, update client copy
			localContents = *remoteServerClipContents
			fmt.Printf("Debug: Client Updated Local Copy From Server\n")

			//Update  OS clipboard
			clipboard.WriteAll(remoteServerClipContents.Contents)
			fmt.Printf("Debug: Client Updated OS Clipboard From Server\n")
		}
	}
	//fmt.Printf("%x\r", hash)
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
			//fmt.Printf(">>>postedContents %s, error: %s\n", postedClipContents, err)
		}
		serverContents = *postedClipContents
		serverContents.UpdateTime = time.Now()

	default:
		http.Error(w, "page not found", http.StatusNotFound)
	}
}
