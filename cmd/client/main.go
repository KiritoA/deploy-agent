package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func exitWithMessage(message string) {
	fmt.Print(message)
	os.Exit(1)
}

func main() {
	_url := flag.String("url", "", "Endpoint url")
	service := flag.String("service", "", "Service name")
	image := flag.String("image", "", "Image name")
	tag := flag.String("tag", "", "Image tag")
	username := flag.String("username", "", "Username")
	password := flag.String("password", "", "password")
	token := flag.String("token", "", "Token")
	flag.Parse()

	if *_url == "" {
		exitWithMessage("Endpoint url required")
	}

	if *service == "" {
		exitWithMessage("Service required")
	}

	if *image == "" {
		exitWithMessage("Image required")
	}

	if *token == "" {
		exitWithMessage("Token required")
	}

	authConfig := types.AuthConfig{
		Username:      *username,
		Password:      *password,
	}

	encodedAuthConfig := ""
	if (types.AuthConfig{} != authConfig) {
		authBytes, _ := json.Marshal(authConfig)
		encodedAuthConfig = base64.URLEncoding.EncodeToString(authBytes)
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST",
		*_url+"/update",
		strings.NewReader(url.Values{
			"service": {*service},
			"image":   {*image},
			"tag":     {*tag},
		}.Encode()))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "Bearer "+*token)
	req.Header.Add("X-Registry-Auth", encodedAuthConfig)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Request failed: ", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Print(string(body))
}
