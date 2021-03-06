# asprika
Deploy tool for docker when use docker swarm.
Asprika connect to swarm manager host, and update service.
If the service is not running in swarm, setup environment variables, migration, and create new service.

## Install
Get binary from github:
```
$ wget https://github.com/h3poteto/asprika/releases/download/v0.1.0/asprika_linxu_amd64.zip
```

or, build. It requires Go1.7 and [gom](https://github.com/mattn/gom).
```
$ git clone git@github.com:h3poteto/asprika.git
$ cd asprika
$ gom install
$ gom build
```

## Setup
Prepare `setting.yml` for deploy configurations, like [sample](https://github.com/h3poteto/asprika/blob/master/setting.yml.sample).
```yaml
user: "root" # User name on remote host for ssh
host: "192.168.0.0" # Remote host name or IP address for ssh
port: ":22" # Remote host port for ssh
docker_image_name: "nginx"  # Docker image name on hub.docker.io
docker_image_tag: "latest" # Docker image tag
container_name: "nginx" # Custom container name
port_forward:
  container_port: 8080
  host_port: 8080
env_file: "/home/ubuntu/.docker-env" # Custom environm nt file for docker
shared_directory:
  source: "/home/ubuntu" # Shared directory in host machine
  target: "/opt" # Shared directoy in docker container
host_name: "example.com" # Host name for curl check
migration: "migrate" # Custom migration command
```

If you don't need shared directory, you can skip this option.

## Run
```
$ ./asprika
```
or
```
$ ./asprika --config custom_setting.yml
```
