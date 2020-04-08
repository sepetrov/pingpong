.PHONY: \
	all \
	build \
	clean \
	create-network \
	image \
	image-client \
	image-consumer \
	image-server \
	start \
	start-client \
	start-consumer \
	start-dd-agent \
	start-server \

include .env

export $(shell [ -f .env ] && sed 's/^\([A-Z_]*\).*/\1/' .env)

DD_SITE?=datadoghq.eu
SERVER_ADDR?=http://pingpong-server:8080

.DEFAULT_GOAL:=help

all: ## Start the whole thing
all: image create-network start


build: ## Build PingPong client, server and consumer
	echo "\nBuilding PingPong client, server and consumer\n"
	mkdir -p ./bin
	go build -o ./bin/pingpong-client ./cmd/client/
	go build -o ./bin/pingpong-client ./cmd/consumer/
	go build -o ./bin/pingpong-server ./cmd/server/

image: ## Build PingPong client, server and consumer Docker images
image: image-client image-consumer image-server

image-client: ## Build PingPong client Docker image
	echo "\nBuilding PingPong client Docker images\n"
	docker build \
		--build-arg 'CMD_PATH=./cmd/client' \
		-f Dockerfile \
		-t pingpong-client \
		.

image-consumer: ## Build PingPong consumer Docker image
	echo "\nBuilding PingPong consumer Docker images\n"
	docker build \
		--build-arg 'CMD_PATH=./cmd/consumer' \
		-f Dockerfile \
		-t pingpong-consumer \
		.

image-server: ## Build PingPong server Docker image
	echo "\nBuilding PingPong server Docker images\n"
	docker build \
		--build-arg 'CMD_PATH=./cmd/server' \
		-f Dockerfile \
		-t pingpong-server \
		.

create-network: ## Create PingPong network
	echo "\nCreating PingPong network\n"
	docker network create pingpong-network

start: ## Start DataDog agent, PingPong client, server and consumer
start: start-dd-agent start-server start-consumer start-client

start-dd-agent: ## Start DataDog agent
	@echo "\nStarting DataDog agent\n"
	@:$(call check_defined, DD_API_KEY, DataDog API key is required)
	DOCKER_CONTENT_TRUST=1 docker run \
		--cap-add=NET_ADMIN \
		--cap-add=SYS_ADMIN \
		--cap-add=SYS_PTRACE \
		--cap-add=SYS_RESOURCE \
		--name dd-agent \
		--network pingpong-network \
		--security-opt apparmor:unconfined \
		-d \
		-e DD_AC_EXCLUDE="name:dd-agent" \
		-e DD_API_KEY=${DD_API_KEY} \
		-e DD_APM_DD_URL=https://trace.agent.${DD_SITE} \
		-e DD_APM_ENABLED=true \
		-e DD_APM_NON_LOCAL_TRAFFIC=true \
		-e DD_LOG_LEVEL=info \
		-e DD_LOGS_CONFIG_CONTAINER_COLLECT_ALL=true \
		-e DD_LOGS_ENABLED=true \
		-e DD_PROCESS_AGENT_ENABLED=true \
		-e DD_SITE=${DD_SITE}  \
		-e DD_SYSTEM_PROBE_ENABLED=true \
		-p 127.0.0.1:8126:8126/tcp \
		-v /opt/datadog-agent/run:/opt/datadog-agent/run:rw \
		-v /proc/:/host/proc/:ro \
		-v /sys/fs/cgroup/:/host/sys/fs/cgroup:ro \
		-v /sys/kernel/debug:/sys/kernel/debug \
		-v /var/run/docker.sock:/var/run/docker.sock:ro \
		datadog/agent:7

start-server: ## Start PingPong server
	@echo "\nStarting PingPong server\n"
	docker run \
		--name pingpong-server \
		--network pingpong-network \
		-d \
		-e AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} \
		-e AWS_REGION=${AWS_REGION} \
		-e AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} \
		-e DD_AGENT_HOST=dd-agent \
		-e HTTP_PORT=8080 \
		-e SQS_QUEUE_URL=${SQS_QUEUE_URL} \
		-p 8080:8080 \
		pingpong-server

start-consumer: ## Start PingPong consumer
	@echo "\nStarting PingPong consumer\n"
	docker run \
		--name pingpong-consumer \
		--network pingpong-network \
		-d \
		-e AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} \
		-e AWS_REGION=${AWS_REGION} \
		-e AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} \
		-e DD_AGENT_HOST=dd-agent \
		-e SQS_QUEUE_URL=${SQS_QUEUE_URL} \
		pingpong-consumer

start-client: ## Start PingPong client
	@echo "\nStarting PingPong client\n"
	docker run \
		--name pingpong-client \
		--network pingpong-network \
		-d \
		-e DD_AGENT_HOST=dd-agent \
		-e SERVER_ADDR=${SERVER_ADDR} \
		pingpong-client

clean: ## Clean up
	-docker rm -fv \
		dd-agent \
		pingpong-client \
		pingpong-consumer \
		pingpong-server
	-docker network remove pingpong-network

##
##  * Help
##

help:    ## Show this help message
	@echo
	@echo '  Usage:'
	@echo '    make <target>'
	@echo
	@echo '  Targets:'
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
	@echo

env:     ## Show exported / environment variables
	@env | sort

#
# Functions
#

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
        $(error Undefined $1$(if $2, ($2))$(if $(value @), \
                required by target `$@')))