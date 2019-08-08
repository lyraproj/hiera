# Hiera lookup framework

## Introduction

Hiera is a flexible, powerful tool for resolving values for variable lookups, which was first popularised by its use in [Puppet](https://puppet.com/docs/puppet/5.5/hiera.html).

This module is a "clean-room" Go implementation of the Hiera framework, suitable for use as a library from other tools.

## Details

Hiera uses the concept of "managing by exception": you design a *hierarchy* of data sources, with the most specific source at the top and  least-specific defaults at the bottom. Hiera searches for keys starting at the top, allowing more-specific sources to override defaults. Sources are usually YAML files stored on the filesystem, and layers usually use variable interpolation to find the right file, allowing the context of the lookup to pick the right file.

## How to run it

### Standalone execution

Hiera is a go module and go modules must be enabled by setting the environment variable GO111MODULE is on before an
attempt is made to install:

    export GO111MODULE=on

#### Install the module

To install the module under $GOPATH/src:

    go get github.com/lyraproj/hiera

#### Install the lookup binary

Install the lookup binary under $GOPATH/bin:

    go install github.com/lyraproj/hiera/lookup

#### Run the binary

    lookup --help

### Containerized execution

#### Download the container

You can pull the latest containerized version from docker hub:

    docker pull lyraproj/hiera:latest

The docker repository with previous tags is viewable at https://hub.docker.com/r/lyraproj/hiera .

#### Run the container

The docker image accepts environment variables to override default behaviour:

* *port* - which port to listen on inside the container (default: 8080)
* *loglevel* - how much logging to do (default: error, possible values: error, warn, info, debug)
* *config* - path to a hiera.yaml configuration (default: /hiera/hiera.yaml)

Make sure to pass the port on your host through to the container. A directory with a hiera configuration and data files (see below) should be mounted under `/hiera` in the image using a bind mount:

    docker run -p 8080:8080 --mount type=bind,src=$HOME/hiera,dst=/hiera lyraproj/hiera:latest

#### Query the container

The web service in the container responds to the `/lookup` endpoint with an additional path element of which key to look up. Nested keys can be looked up using dot-separation notation. Given a yaml map without any overrides like:

    aws:
      tags:
        Name:       lyra-sample
        created_by: lyra
        department: engineering
        project:    incubator
        lifetime:   1h

You can get back the entire map or specific parts of it:

    $ curl http://localhost:8080/lookup/aws
    {"tags":{"Name":"lyra-sample","created_by":"lyra","department":"engineering","lifetime":"1h","project":"incubator"}}
    $ curl http://localhost:8080/lookup/aws.tags
    {"Name":"lyra-sample","created_by":"lyra","department":"engineering","lifetime":"1h","project":"incubator"}
    $ curl http://localhost:8080/lookup/aws.tags.department
    "engineering"

## Pass values for interpolation

If your hierarchy config contains variable interpolation, you can provide context for the lookup using the `var` query parameter. Repeated `var` parameters will create an array of available parameters. The values should be colon-separated variable-value pairs:

    curl 'http://localhost:8080/lookup/aws.tags?var=environment:production&var=hostname:specialhost'

TODO: Nested variable lookups such like `os.family` are not yet working.

## Hiera configuration and directory structure

Much of hiera's power lies in its ability to interpolate variables in the hierarchy's configuration. A lookup provides values, and hiera maps the interpolated values onto the filesystem (or other back-end data structure). A common example uses two levels of override: one for specific hosts, a higher layer for environment-wide settings, and finally a fall-through default. A functional `hiera.yaml` which implements this policy looks like:

    ---
    version: 5
    defaults:
    datadir: hiera
    data_hash: yaml_data

    hierarchy:
    - name: "Host-specific overrides"
        path: "hosts/%{hostname}.yaml"
    - name: "Environmental overrides"
        path: "environments/%{environment}.yaml"
    - name: "Fall through defaults"
        path: "defaults.yaml"

This maps to a directory structure based in the `hiera` subdirectory (due to the `datadir` top level key) containing yaml files like:

    hiera
    ├── defaults.yaml
    ├── environments
    │   └── production.yaml
    └── hosts
        └── specialhost.yaml

## Azure Key Vault lookup key

This function allows you to look up single values stored as secrets from an Azure Key Vault.
A single option `vault_name` should be set in `hiera.yaml`:

    ---
    version: 5
    defaults:
    datadir: hiera
    data_hash: yaml_data

    hierarchy:
    - name: common
      path: common.yaml
    - name: secrets
      lookup_key: azure_key_vault
      options:
        vault_name: my-key-vault

There are two options for authentication, using a service principal or the Azure CLI

* To use a service principal set the environment variables `AZURE_TENANT_ID`, `AZURE_CLIENT_ID` and `AZURE_CLIENT_SECRET`

* If the above variables are not present the Azure CLI will be used (it must already be logged in)

## Implementation status

* [x] lookup CLI
* [x] lookup function
* [x] lookup context
* [x] dotted keys (dig functionality)
* [x] interpolation using scope, lookup/hiera, alias, or literal function
* [x] Hiera version 5 configuration in hiera.yaml
* [x] merge strategies (first, unique, hash, deep)
* [x] YAML data
* [x] JSON data
* [x] lookup options stored adjacent to data
* [x] convert_to type coercions
* [x] Sensitive data
* [ ] configurable deep merge
* [ ] pluggable back ends (initially for secrets management)
* [x] `explain` functionality to show traversal
* [x] containerized REST-based microservice
* [x] JSON and YAML schema for the hiera.yaml config file (see schema/hiera_v5.yaml)
