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
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/atotto/clipboard"
	"github.com/briandowns/spinner"
	"github.com/codegangsta/cli"
)

type clipContents struct {
	Contents   string
	Hash       [20]byte
	UpdateTime time.Time
}

const defaultPort = "7569"

var defaultServer string
var server string
var verbose bool
var isClientOnly bool
var localContents clipContents
var serverContents clipContents

func main() {

	defaultServer = getLocalIPAddress() + ":" + defaultPort
	app := cli.NewApp()
	app.Name = "clipsync"
	app.Usage = "Synchronizes clipboards between machines. Run as Server on one machine, run as client on another"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "server, s", Value: defaultServer, Usage: "Hostname or IP of server clipsync process and port number.  Use default for server, or specify an IP to run as client", Destination: &server},
		cli.BoolFlag{Name: "verbose, V", Usage: "Verbose output", Destination: &verbose},
		cli.BoolFlag{Name: "isClientOnly, c", Usage: "Sync Client only (must specify remote server)", Destination: &isClientOnly},
	}
	app.Action = func(c *cli.Context) {
		runApp()
	}

	app.Run(os.Args)

}

func runApp() {
	fmt.Printf("Press enter to quit\n")
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)

	if !verbose {
		log.SetOutput(ioutil.Discard)
		s.Start()
	}

	//Kick off the server process
	if !isClientOnly {
		fmt.Printf("Running clipSync server on: %s\n", server)
		http.HandleFunc("/", handleServerRequest)
		log.Printf("...server: %s", server)
		go http.ListenAndServe(server, nil)
	}

	//Kick off the sync process
	endLoop := pushLoop()

	//Block until done.
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
	endLoop <- struct{}{} //send literal of type struct{}

	s.Stop()
}

func pushLoop() chan struct{} {
	ticker := time.NewTicker(500 * time.Millisecond)
	quit := make(chan struct{})

	log.Printf("Syncing clipboard with server %s", server)

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
		log.Printf("Debug: Client Updated Local Copy From OS\n")
	}

	remoteServerClipContents := new(clipContents)

	//Get clipsync server's copy
	resp, err := http.Get("http://" + server)

	if err != nil {
		log.Printf("Error contacting server: %s, error: %s\n", server, err.Error())
	} else {
		err := json.NewDecoder(resp.Body).Decode(remoteServerClipContents)
		if err != nil {
			log.Printf("error: %s\n", err)
		}
		//fmt.Printf(">>>>>>>>>>>%s", remoteServerClipContents) //TODO MJT, not getting good stuff here.
	}

	log.Printf("local: %s, server %s\n", localContents.Hash, remoteServerClipContents.Hash)

	//if clipsync client copy != clipsync server copy...
	if localContents.Hash != remoteServerClipContents.Hash {
		if remoteServerClipContents == nil || localContents.UpdateTime.Sub(remoteServerClipContents.UpdateTime) > 1 {

			log.Printf("Need to post \n%s\n\n", remoteServerClipContents.Contents)

			//if server has nothing or clipsync client copy newer, post to server
			_ = "breakpoint"
			byteArray, _ := json.Marshal(localContents)
			reader := bytes.NewReader(byteArray)
			if err == nil {
				req, _ := http.NewRequest("POST", "http://"+server, reader)
				req.Header.Add("content-type", "application/json")
				http.DefaultClient.Do(req)
				log.Printf("Debug: Client Pushed Clipboard to Server %s ", server)
			}

			//Do post.

		} else {
			//clipsync server copy newer, update client copy
			localContents = *remoteServerClipContents
			log.Printf("Debug: Client Updated Local Copy From Server\n")

			//Update  OS clipboard
			clipboard.WriteAll(remoteServerClipContents.Contents)
			log.Printf("Debug: Client Updated OS Clipboard From Server\n")
		}
	}
	//log.Printf("%x\r", hash)
}

func handleServerRequest(w http.ResponseWriter, r *http.Request) {
	_ = "breakpoint"
	switch r.Method {
	case "GET":
		log.Printf("Debug - Server, get")
		json.NewEncoder(w).Encode(serverContents)
	case "POST":
		log.Printf("Debug - Server, post")
		postedClipContents := new(clipContents)
		err := json.NewDecoder(r.Body).Decode(postedClipContents)
		if err != nil {
			log.Printf("Debug: Server - posted contents: %s", postedClipContents)
		} else {
			log.Printf("Debug: Server - postedContents %s, error: %s\n", postedClipContents, err)
		}
		serverContents = *postedClipContents
		serverContents.UpdateTime = time.Now()

	default:
		http.Error(w, "page not found", http.StatusNotFound)
	}
}

func getLocalIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
