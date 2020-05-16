package main

import (
	"crypto/subtle"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

type Config struct {
	Address     string
	Registry    string
	Token       string
}

var config = Config{}

var tokenRegex = regexp.MustCompile(`Bearer\s([a-zA-z0-9]+)`)

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func loadConfig() {
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
		Address:     *address,
		Registry:    *registry,
		Token:       *token,
	}
}

func main() {
	loadConfig()

	log.Println("Starting deploy agent")

	http.HandleFunc("/update", update)
	err := http.ListenAndServe(config.Address, nil)
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
		return
	}

	serviceName := request.FormValue("service")
	if serviceName == "" {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("missing parameter [service]"))
		return
	}

	image := request.FormValue("image")
	if image == "" {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("missing parameter [image]"))
		return
	}

	tag := request.FormValue("tag")

	imageName := config.Registry + "/" + image
	if tag != "" {
		imageName += ":" + tag
	}

	// docker service update --image [Service image tag] --with-registry-auth [Service name]
	cmd := exec.Command("docker", "service", "update",
		"--image", imageName,
		"--with-registry-auth",
		serviceName)
	out, err := cmd.CombinedOutput()
	outputStr := string(out)
	if err != nil {
		errStr := fmt.Sprintf("Docker command execution failed: \n%v\n%v", outputStr, err)
		log.Error(errStr)

		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("Deploy failed: \n%v%v", outputStr, err)))
		return
	}

	_, _ = writer.Write(out)

	log.WithField("service", serviceName).WithField("image", imageName).Info("Deploy completed")
}
