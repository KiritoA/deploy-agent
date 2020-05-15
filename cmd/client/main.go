package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
)

func main() {
	_url := flag.String("url", "", "Endpoint url")
	service := flag.String("service", "", "Service name")
	token := flag.String("token", "", "Token")
	image := flag.String("image", "", "Image name")
	tag := flag.String("tag", "", "Image tag")
	flag.Parse()

	if *_url == "" {
		log.Fatal("Endpoint url required")
	}

	if *service == "" {
		log.Fatal("Service required")
	}

	if *image == "" {
		log.Fatal("Service required")
	}

	if *token == "" {
		log.Fatal("Token required")
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", *_url+"/update", nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "Bearer "+*token)
	req.Form = url.Values{
		"service": {*service},
		"image":   {*image},
		"tag":     {*tag},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Request failed: ", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	log.Info(string(body))
}
