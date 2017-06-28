package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	var passphrase, page, server string
	var encrypt, store bool
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "server",
			Value:       "https://cowyo.com",
			Usage:       "server to use",
			Destination: &server,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "upload",
			Aliases: []string{"u"},
			Usage:   "upload document",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "encrypt, e",
					Usage:       "encrypt using passphrase",
					Destination: &encrypt,
				},
				cli.BoolFlag{
					Name:        "store, s",
					Usage:       "store and persist after reading",
					Destination: &store,
				},
				cli.StringFlag{
					Name:        "passphrase, a",
					Usage:       "passphrase to use for encryption",
					Destination: &passphrase,
				},
				cli.StringFlag{
					Name:        "page, p",
					Usage:       "specific page to use",
					Destination: &page,
				},
			},
			Action: func(c *cli.Context) error {
				var data []byte
				var err error
				if c.NArg() == 0 {
					data, err = ioutil.ReadAll(os.Stdin)
					if err != nil {
						return err
					}
					log.Printf("stdin data: %v\n", string(data))
				} else {
					data, err = ioutil.ReadFile(c.Args().Get(0))
					if err != nil {
						return err
					}
					log.Printf("file data: %v\n", string(data))
				}
				return uploadData(server, page, string(data), encrypt, passphrase, store)
			},
		},
		{
			Name:    "download",
			Aliases: []string{"d"},
			Usage:   "download document",
			Action: func(c *cli.Context) error {
				page := ""
				if c.NArg() == 1 {
					page = c.Args().Get(0)
				} else {
					return errors.New("Must specify page")
				}
				return downloadData(server, page)
			},
		},
	}

	errMain := app.Run(os.Args)
	if errMain != nil {
		log.Println(errMain)
	}

}

func uploadData(server string, page string, text string, encrypt bool, passphrase string, store bool) (err error) {
	if page == "" {
		// generate page name
		page = "foo12"
	}
	if encrypt {
		log.Println("Encryption activated")
		if passphrase == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter passphrase: ")
			passphrase, _ = reader.ReadString('\n')
			passphrase = strings.TrimSpace(passphrase)
		}
		text, err = EncryptString(text, passphrase)
		if err != nil {
			return err
		}
	}

	type Payload struct {
		NewText     string `json:"new_text"`
		Page        string `json:"page"`
		IsEncrypted bool   `json:"is_encrypted"`
		IsPrimed    bool   `json:"is_primed"`
	}

	data := Payload{
		NewText:     text,
		Page:        page,
		IsEncrypted: encrypt,
		IsPrimed:    !store,
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", server+"/update", body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var target interface{}
	json.NewDecoder(resp.Body).Decode(&target)
	log.Printf("%v", target)
	return
}

func downloadData(server string, page string) (err error) {
	type Payload struct {
		Page string `json:"page"`
	}

	data := Payload{
		Page: page,
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", server+"/relinquish", body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	type Response struct {
		Destroyed bool   `json:"destroyed"`
		Encrypted bool   `json:"encrypted"`
		Locked    bool   `json:"locked"`
		Message   string `json:"message"`
		Success   bool   `json:"success"`
		Text      string `json:"text"`
	}
	var target Response
	json.NewDecoder(resp.Body).Decode(&target)
	log.Printf("%v", target)
	return
}
