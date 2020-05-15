package main

import (
	"bytes"
	"crypto/subtle"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

type Config struct {
	Address     string
	ComposePath string
	WorkingDir  string
	Registry    string
	Mode        string
	Token       string
	projectName string
}

type CommandParams struct {
	Name string
	Args []string
}

var config = Config{}

const ModeCompose = "compose"
const ModeStack = "stack"

var tokenRegex = regexp.MustCompile(`Bearer\s([a-zA-z0-9]+)`)

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func getTestCommandParams(config Config) (params CommandParams) {
	switch config.Mode {
	case ModeCompose:
		params.Name = "docker-compose"
		params.Args = []string{"-f", config.ComposePath}

		if config.projectName != "" {
			params.Args = append(params.Args, "-p")
			params.Args = append(params.Args, config.projectName)
		}

		params.Args = append(params.Args, "ps")
		break
	case ModeStack:
		params.Name = "docker"
		params.Args = []string{"stack", "ps", config.projectName}
	}
	return
}

func getDeployCommandParams(config Config) (params CommandParams) {
	switch config.Mode {
	case ModeCompose:
		params.Name = "docker-compose"
		params.Args = []string{"-f", "-"}

		if config.projectName != "" {
			params.Args = append(params.Args, "-p")
			params.Args = append(params.Args, config.projectName)
		}

		params.Args = append(params.Args, "up")
		params.Args = append(params.Args, "-d")
		break
	case ModeStack:
		params.Name = "docker"
		params.Args = []string{"stack", "deploy", "-c", "-", "--with-registry-auth", config.projectName}
	}
	return
}

func loadConfig() {
	address := flag.String("address", ":8090", "Server listen address")
	composePath := flag.String("compose-file", "", "Path to a Compose file")
	workingDir := flag.String("working-dir", "", "Working directory")
	registry := flag.String("registry", "", "Registry url")
	mode := flag.String("mode", "compose", "Either `compose` or `stack`")
	projectName := flag.String("project", "", "Project name (compose) or stack name (Swarm)")
	token := flag.String("token", "", "Authorization token")
	flag.Parse()

	config.Address = *address

	if *composePath == "" {
		log.Fatal("Missing option [compose-file]")
	}
	config.ComposePath = *composePath

	if *workingDir != "" {
		if _, err := os.Stat(*workingDir); os.IsNotExist(err) {
			log.Fatalf("Working directory [%v] doesn't exists", *workingDir)
		}
		config.WorkingDir = *workingDir
	}

	if *registry == "" {
		log.Fatalf("Registry url must not be empty")
	}
	config.Registry = *registry

	if *mode != ModeCompose && *mode != ModeStack {
		log.Fatalf("Invalid mode [%v]", mode)
	}
	config.Mode = *mode

	config.projectName = *projectName

	if *token == "" {
		log.Fatalf("Missing token argument")
	}

	if len(*token) < 16 {
		log.Fatalf("Token must be at least 16 bytes")
	}
	config.Token = *token
}

func main() {
	loadConfig()

	log.Println("Starting deploy agent")

	log.Debug("Running compose file test")

	params := getTestCommandParams(config)
	cmd := exec.Command(params.Name, params.Args...)
	cmd.Dir = config.WorkingDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Docker command test failed: \n%v\n%v", string(out), err)
	}

	log.Debug("Compose file test passed")

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

	data, err := ioutil.ReadFile(config.ComposePath)

	// Parse compose file
	var composeConfigMap map[string]interface{}
	err = yaml.Unmarshal([]byte(data), &composeConfigMap)
	if err != nil {
		errStr := fmt.Sprintf("error: %v", err)
		log.Error(errStr)
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(errStr))
		return
	}

	_services, ok := composeConfigMap["services"]
	if !ok {
		errStr := "No services field found in compose file"
		log.Error(errStr)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(errStr))
		return
	}

	services := _services.(map[interface{}]interface{})
	if !ok {
		errStr := "Invalid services structure"
		log.Error(errStr)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(errStr))
		return
	}

	imageName := config.Registry + "/" + image
	if tag != "" {
		imageName += ":" + tag
	}

	found := false
	for serviceFieldName, value := range services {
		if serviceFieldName == serviceName {
			serviceItem, ok := value.(map[interface{}]interface{})
			if !ok {
				errStr := "Invalid service item structure"
				log.Error(errStr)
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte(errStr))
				return
			}

			serviceItem["image"] = imageName
			found = true
			break
		}
	}

	if !found {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("Invalid service [%s]", serviceName)))
		return
	}

	yamlBytes, err := yaml.Marshal(composeConfigMap)
	if err != nil {
		errStr := fmt.Sprintf("Yaml marshal failed: %v", err)
		log.Error(errStr)
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(errStr))

		return
	}

	params := getDeployCommandParams(config)
	cmd := exec.Command(params.Name, params.Args...)
	cmd.Stdin = bytes.NewReader(yamlBytes)
	cmd.Dir = config.WorkingDir
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
