package main

import (
	"io/ioutil"
	"log"

	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"
	_ "golang.org/x/crypto/ssh/agent"
	"gopkg.in/yaml.v2"
)

type Deploy struct {
	User string
	Host string
	Port string

	DockerImageName string
	DockerImageTag  string
	ContainerName   string
	ContainerPort   int

	SharedDirectory string

	HostName string

	Client *ssh.Client
}

func main() {
	configFile := flag.StringP("config", "c", "setting.yml", "Config file")
	flag.Parse()

	deploy, err := initialize(configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = deploy.deploy()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Deploy success")
}

func initialize(configFile *string) (*Deploy, error) {
	buf, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	m := make(map[interface{}]interface{})
	err = yaml.Unmarshal(buf, &m)
	if err != nil {
		return nil, err
	}
	d := &Deploy{
		User:            m["user"].(string),
		Host:            m["host"].(string),
		Port:            m["port"].(string),
		DockerImageName: m["docker_image_name"].(string),
		DockerImageTag:  m["docker_image_tag"].(string),
		ContainerName:   m["container_name"].(string),
		ContainerPort:   m["container_port"].(int),
		SharedDirectory: m["shared_directory"].(string),
		HostName:        m["host_name"].(string),
	}
	return d, nil
}
