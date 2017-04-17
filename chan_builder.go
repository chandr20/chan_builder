package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"encoding/base64"
	"encoding/json"
	"net/http/httputil"
)

type PassedParams struct {
	Image_name string
	Username   string
	Password   string
	Email      string
	Dockerfile string
}

type PushAuth struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	Serveraddress string `json:"serveraddress"`
	Email         string `json:"email"`
}

type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func main() {

	passedparams := PassedParams{}

	Buildparams(&passedparams)

	fmt.Println(passedparams.Image_name)

	BuildPushDeleteImage(passedparams)
}

func Buildparams(passedparams *PassedParams) *PassedParams {

	Image_name := os.Getenv("Image_name")
	Username := os.Getenv("Username")
	Password := os.Getenv("Password")
	Email := os.Getenv("Email")
	Dockerfile := os.Getenv("Dockerfile")

	*passedparams = PassedParams{
		Image_name: Image_name,
		Username:   Username,
		Password:   Password,
		Email:      Email,
		Dockerfile: Dockerfile,
	}
	return passedparams

}

func BuildPushDeleteImage(passedParams PassedParams) {
	splitImageName := make([]string, 2)
	fmt.Println(passedParams.Image_name)
	fmt.Println(passedParams.Dockerfile)

	if strings.Contains(passedParams.Image_name, ".") {
		splitImageName := strings.SplitN(passedParams.Image_name, "/", 2)
		fmt.Println(splitImageName)

	}
	buildUrl := ("/v1.28/build?nocache=true&t=" + passedParams.Image_name)
	dockerDial := Dial()
	dockerConnection := httputil.NewClientConn(dockerDial, nil)
	readerForInput, err := ReaderForInputType(passedParams)

	if err != nil {
		log.Println(err)
	}
	fmt.Println(buildUrl, readerForInput)
	buildreq, err := http.NewRequest("POST", buildUrl, readerForInput)
	fmt.Println(StringEncAuth(passedParams, ServerAddress(splitImageName[0])))

	buildreq.Header.Add("X-Registry-Config", StringEncAuth(passedParams, ServerAddress(splitImageName[0])))
	buildresponse, err := dockerConnection.Do(buildreq)
	fmt.Println(buildresponse.Status)
	fmt.Println(buildresponse.Body)
	defer buildresponse.Body.Close()
	if err != nil {
		log.Println(err)
	}

}

func Dial() net.Conn {

	var docker_proto string
	var docker_host string
	if os.Getenv("DOCKER_HOST") != "" {
		docker_host := os.Getenv("DOCKER_HOST")
		splitstrings := strings.SplitN(docker_host, "://", 2)
		docker_proto = splitstrings[0]
		docker_host = splitstrings[1]

	} else {
		docker_proto = "tcp"
		docker_host = "localhost:4243"
	}

	docker_dial, err := net.Dial(docker_proto, docker_host)
	if err != nil {
		log.Println("Failed to reach docker")
		log.Fatal(err)

	}

	return docker_dial

}

func ReaderForInputType(passedparams PassedParams) (io.Reader, error) {
	switch {

	case passedparams.Dockerfile != "":
		return ReaderForDockerfile(passedparams.Dockerfile), nil

	default:
		return nil, errors.New("Failed in the ReaderForInputType.  Got to default case.")
	}

}

func ReaderForDockerfile(dockerfile string) *bytes.Buffer {
	//create a buffer to write our archieve
	fmt.Println(dockerfile)
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", dockerfile},
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Size: int64(len(file.Body)),
		}

		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatalln(err)

		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			log.Fatalln(err)
		}

	}
	if err := tw.Close(); err != nil {
		log.Fatalln(err)
	}

	return buf
}

func ServerAddress(privaterepo string) string {
	var serveraddress string

	if privaterepo != "" {

		serveraddress = ("http://" + privaterepo + "v1")

	} else {
		serveraddress = ("https://index.docker.io/v1/")
	}
	return serveraddress

}

func StringEncAuth(passedparams PassedParams, serveraddress string) string {
	var data Auth
	data.Username = passedparams.Username
	data.Password = passedparams.Password
	data.Email = passedparams.Email
	config := make(map[string]Auth)
	config[serveraddress] = data
	json_data, err := json.Marshal(config)
	if err != nil {
		log.Println(err)
	}
	sEnc := base64.StdEncoding.EncodeToString([]byte(json_data))
	return sEnc

}
