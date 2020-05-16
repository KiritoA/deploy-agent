package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"os"
)

var dockerClient *client.Client

func init() {
	var err error
	dockerClient, err = client.NewEnvClient()
	if err != nil {
		log.Fatalf("Failed to initialize docker client: %v", err)
	}
}

func main() {
	registry := flag.String("registry", "", "Registry url")
	username := flag.String("username", "", "Username")
	password := flag.String("password", "", "password")
	flag.Parse()

	if *registry == "" {
		fmt.Println("Registry url must not be empty")
		os.Exit(1)
	}

	if *username == "" {
		fmt.Println("Username must not be empty")
		os.Exit(1)
	}

	if *password == "" {
		fmt.Println("password must not be empty")
		os.Exit(1)
	}

	authConfig := types.AuthConfig{
		Username:      *username,
		Password:      *password,
		ServerAddress: *registry,
	}

	resp, err := dockerClient.RegistryLogin(context.Background(), authConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if resp.IdentityToken != "" {
		fmt.Print(resp.IdentityToken)
	} else {
		authBytes, _ := json.Marshal(authConfig)
		authBase64 := base64.URLEncoding.EncodeToString(authBytes)
		fmt.Print(authBase64)
	}
}
