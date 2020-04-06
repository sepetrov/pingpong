.PHONY: \
	start-dd-agent \


include .env

export $(shell [ -f .env ] && sed 's/^\([A-Z_]*\).*/\1/' .env)

DD_SITE=datadoghq.eu

export DD_SITE


.DEFAULT_GOAL:=help

start-dd-agent: ## Start DataDog agent
	@echo "\nStarting DataDog agent\n"
	@:$(call check_defined, DD_API_KEY, DataDog API key is required)
	@if [ $(shell docker ps --format '{{.Names}}' | grep dd-agent | wc -l) == "0" ]; then \
		DOCKER_CONTENT_TRUST=1 docker run \
			-d \
			--name dd-agent \
			-v /var/run/docker.sock:/var/run/docker.sock:ro \
			-v /proc/:/host/proc/:ro \
			-v /sys/fs/cgroup/:/host/sys/fs/cgroup:ro \
			-e DD_API_KEY=${DD_API_KEY} \
			-e DD_SITE=${DD_SITE} \
			datadog/agent:7 \
			; \
	fi

up: ## Start services
up: start-dd-agent
	@echo "\nStarting PingPong service\n"
	docker-compose up --build -d --force-recreate --remove-orphans

clean: ## Stop services
	@echo "\nCleaning up\n"
	-docker stop dd-agent
	-docker rm -v dd-agent
	-docker-compose stop
	-docker-compose rm -sfv

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