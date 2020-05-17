package main

import (
	"context"
	"crypto/subtle"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"regexp"
)

type Config struct {
	Address  string
	Registry string
	Token    string
}

var config = Config{}
var tokenRegex = regexp.MustCompile(`Bearer\s([a-zA-z0-9]+)`)
var dockerClient *client.Client

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)

	var err error
	dockerClient, err = client.NewEnvClient()
	if err != nil {
		log.Fatalf("Failed to initialize docker client: %v", err)
	}
}

func main() {
	address := flag.String("address", ":8090", "Server listen address")
	registry := flag.String("registry", "", "Trusted registry")
	token := flag.String("token", "", "Authorization token")
	flag.Parse()

	if *registry == "" {
		log.Fatalf("Registry url must not be empty")
	}

	if *token == "" {
		log.Fatalf("Missing token argument")
	}

	if len(*token) < 16 {
		log.Fatalf("Token must be at least 16 bytes")
	}

	config = Config{
		Address:  *address,
		Registry: *registry,
		Token:    *token,
	}

	_, err := dockerClient.SwarmInspect(context.Background())
	if err != nil {
		log.Fatal("Failed to connect to docker: ", err)
	}

	log.Printf("Starting deploy agent at [%s]", config.Address)

	http.HandleFunc("/update", update)
	err = http.ListenAndServe(config.Address, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func isAuthorized(request *http.Request) bool {
	_token := tokenRegex.FindStringSubmatch(request.Header.Get("Authorization"))
	if len(_token) <= 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(_token[1]), []byte(config.Token)) == 1
}

func update(writer http.ResponseWriter, request *http.Request) {
	if !isAuthorized(request) {
		writer.WriteHeader(http.StatusUnauthorized)
		writer.Write([]byte("Unauthorized"))
		return
	}

	serviceName := request.FormValue("service")
	if serviceName == "" {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte("missing parameter [service]"))
		return
	}

	image := request.FormValue("image")
	if image == "" {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte("missing parameter [image]"))
		return
	}

	tag := request.FormValue("tag")

	imageName := config.Registry + "/" + image
	if tag != "" {
		imageName += ":" + tag
	}

	logger := log.WithField("service", serviceName).WithField("image", imageName)

	service, _, err := dockerClient.ServiceInspectWithRaw(context.Background(), serviceName)
	if err != nil {
		if client.IsErrServiceNotFound(err) {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte(fmt.Sprintf("Invalid service [%v]", serviceName)))
		} else {
			logger.Error("Failed to inspect service: ", err)
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte(fmt.Sprintf("Failed to inspect service: %v", err)))
		}
		return
	}

	serviceSpec := &service.Spec
	serviceSpec.TaskTemplate.ContainerSpec.Image = imageName

	updateResp, err := dockerClient.ServiceUpdate(context.Background(), serviceName, service.Version, *serviceSpec,
		types.ServiceUpdateOptions{EncodedRegistryAuth: request.Header.Get("X-Registry-Auth")})
	if err != nil {
		logger.Error("Failed to update service: ", err)
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(fmt.Sprintf("Failed to update service: %v", err)))
		return
	}

	if len(updateResp.Warnings) > 0 {
		writer.Write([]byte("Warnings:\n"))

		for _, warn := range updateResp.Warnings {
			writer.Write([]byte(warn))
		}
	}

	logger.Info("Update completed")
}
