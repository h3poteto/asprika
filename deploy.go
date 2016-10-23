package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// 新しいdocker imageをpullする
// db:migrate
// serviceが生きているか確認する
// 既に動いている場合には，service updateをかける
// 動いていなかった場合には新規に立ち上げる
// 待つ
// 念の為curlする
// 古いdockerコンテナを削除する
// 古いdocker imageを削除する

func (d *Deploy) deploy() error {
	err := d.prepareDockerImage()
	if err != nil {
		return err
	}

	err = d.migration()
	if err != nil {
		return err
	}

	alive, err := d.checkRunningService()
	if alive {
		err = d.serviceUpdate()
		if err != nil {
			return err
		}

		// update-dely + stop-grace-period分だけ待つ
		log.Println("Waiting service update...")
		time.Sleep((30 + 20) * time.Second)
	} else {
		log.Println(err)
		err = d.serviceCreate()
		if err != nil {
			return err
		}

		log.Println("Waiting service create...")
		time.Sleep(5 * time.Second)
	}

	_, err = d.checkServiceLiving()
	if err != nil {
		return err
	}

	err = d.removeOldContainer()
	if err != nil {
		log.Println(err)
	}

	err = d.removeOldImages()
	if err != nil {
		log.Println(err)
	}

	return nil
}

func (d *Deploy) initClient() {
	user := d.User

	sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		panic(err)
	}
	signers, err := agent.NewClient(sock).Signers()
	if err != nil {
		panic(err)
	}
	auths := []ssh.AuthMethod{ssh.PublicKeys(signers...)}
	config := &ssh.ClientConfig{
		User: user,
		Auth: auths,
	}
	d.Client, err = ssh.Dial("tcp", d.Host+d.Port, config)
	if err != nil {
		panic(err)
	}
}

func (d *Deploy) getSession() *ssh.Session {
	if d.Client == nil {
		d.initClient()
	}

	session, err := d.Client.NewSession()
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	if err != nil {
		panic(err)
	}

	return session
}

func (d *Deploy) prepareDockerImage() error {
	session := d.getSession()
	defer session.Close()

	log.Println("Docker pull...")
	command := fmt.Sprintf("docker pull %v:%v", d.DockerImageName, d.DockerImageTag)
	log.Println(command)
	if err := session.Run(command); err != nil {
		return err
	}
	return nil
}

func (d *Deploy) migration() error {
	session := d.getSession()
	defer session.Close()

	log.Println("db migration...")
	command := fmt.Sprintf("docker run --rm --env-file %s %s %s", d.EnvFile, d.DockerImageName, d.Migration)
	log.Println(command)
	if err := session.Run(command); err != nil {
		return err
	}
	return nil
}

func (d *Deploy) checkRunningService() (bool, error) {
	session := d.getSession()
	defer session.Close()

	session.Stdout = nil
	session.Stderr = nil

	log.Println("Check running docker service...")
	command := fmt.Sprintf("docker service ls -q --filter name=%s", d.ContainerName)
	log.Println(command)
	result, err := session.CombinedOutput(command)
	log.Println(string(result))
	if err != nil {
		return false, err
	}
	if len(result) > 0 {
		return true, nil
	}
	return false, errors.New("Service is not alive")
}

func (d *Deploy) serviceCreate() error {
	// memo: 現状docker service createにはenf-file指定ができないので，サーバ内のenvfileをparseしてenvとして渡す
	environments, err := d.parseEnvfile()
	if err != nil {
		return err
	}

	envOptions := ""
	for _, e := range *environments {
		option := "--env " + e + " "
		envOptions += option
	}

	session := d.getSession()
	defer session.Close()

	log.Println("Service create...")
	command := fmt.Sprintf("docker service create --publish %d:%d --name %s --replicas 2 --update-delay 20s --stop-grace-period 10s --mount type=bind,source=%s,target=%s,readonly=false %s %s:%s", d.PortForward.HostPort, d.PortForward.ContainerPort, d.ContainerName, d.SharedDirectory.Source, d.SharedDirectory.Target, envOptions, d.DockerImageName, d.DockerImageTag)
	log.Println(command)
	if err := session.Run(command); err != nil {
		return err
	}
	return nil
}

func (d *Deploy) serviceUpdate() error {
	session := d.getSession()
	defer session.Close()

	log.Println("Service update...")
	command := fmt.Sprintf("docker service update --image %s:%s %s", d.DockerImageName, d.DockerImageTag, d.ContainerName)
	log.Println(command)
	if err := session.Run(command); err != nil {
		return err
	}
	return nil
}

func (d *Deploy) checkServiceLiving() (int, error) {
	session := d.getSession()
	defer session.Close()

	session.Stdout = nil
	session.Stderr = nil

	log.Println("Check service is living...")
	command := fmt.Sprintf("curl --insecure -H '%v' https://127.0.0.1 -o /dev/null -w '%%{http_code}' -s", d.HostName)
	log.Println(command)
	result, err := session.CombinedOutput(command)
	if err != nil {
		return 0, err
	}
	if len(result) <= 0 {
		return 0, errors.New("Can not get HTTP status code")
	}
	statusCode, err := strconv.Atoi(string(result))
	if err != nil {
		return 0, err
	}
	if statusCode != 200 {
		return statusCode, errors.New("HTTP status code is not 200")
	}
	return statusCode, nil
}

func (d *Deploy) removeOldContainer() error {
	session := d.getSession()
	defer session.Close()

	log.Println("Remove old docker container")
	command := "docker rm `docker ps -a -q`"
	log.Println(command)
	if err := session.Run(command); err != nil {
		return err
	}

	return nil
}

func (d *Deploy) removeOldImages() error {
	session := d.getSession()
	defer session.Close()

	log.Println("Remove old docker images")
	command := fmt.Sprintf("docker rmi -f $(docker images | awk '/<none>/ { print $3 }')")
	log.Println(command)
	if err := session.Run(command); err != nil {
		return err
	}
	return nil
}

func (d *Deploy) parseEnvfile() (*[]string, error) {
	session := d.getSession()
	defer session.Close()

	session.Stdout = nil
	session.Stderr = nil

	command := fmt.Sprintf("cat %s", d.EnvFile)
	log.Println(command)
	result, err := session.CombinedOutput(command)
	if err != nil {
		return nil, err
	}

	var environments []string
	for _, s := range strings.Split(string(result), "\n") {
		if len(s) > 1 {
			environments = append(environments, s)
		}
	}
	return &environments, err
}
