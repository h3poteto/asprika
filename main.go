package main

import (
	"errors"
	"io/ioutil"
	"log"

	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"
	_ "golang.org/x/crypto/ssh/agent"
	"gopkg.in/yaml.v2"
)

type sharedDirectory struct {
	Source string
	Target string
}

type portForward struct {
	ContainerPort int
	HostPort      int
}

type Deploy struct {
	User string
	Host string
	Port string

	DockerImageName string
	DockerImageTag  string
	ContainerName   string
	PortForward     *portForward

	EnvFile         string
	SharedDirectory *sharedDirectory

	HostName  string
	Protocol  string
	Migration string

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

	p := &portForward{
		ContainerPort: m["port_forward"].(map[interface{}]interface{})["container_port"].(int),
		HostPort:      m["port_forward"].(map[interface{}]interface{})["host_port"].(int),
	}

	d := &Deploy{
		User:            m["user"].(string),
		Host:            m["host"].(string),
		Port:            m["port"].(string),
		DockerImageName: m["docker_image_name"].(string),
		DockerImageTag:  m["docker_image_tag"].(string),
		ContainerName:   m["container_name"].(string),
		PortForward:     p,
		EnvFile:         m["env_file"].(string),
		SharedDirectory: nil,
		HostName:        m["host_name"].(string),
		Protocol:        "http",
		Migration:       m["migration"].(string),
	}

	if m["shared_directory"] != nil {
		s := &sharedDirectory{
			Source: m["shared_directory"].(map[interface{}]interface{})["source"].(string),
			Target: m["shared_directory"].(map[interface{}]interface{})["target"].(string),
		}
		d.SharedDirectory = s
	}

	if m["protocol"] != nil {
		if m["protocol"].(string) != "http" && m["protocol"].(string) != "https" {
			return nil, errors.New("Unknown protocol")
		}
		d.Protocol = m["protocol"].(string)
	}

	return d, nil
}
