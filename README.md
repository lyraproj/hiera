# Hiera lookup framework

[![](https://goreportcard.com/badge/github.com/lyraproj/hiera)](https://goreportcard.com/report/github.com/lyraproj/hiera)
[![](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/lyraproj/hiera)
[![](https://github.com/lyraproj/hiera/workflows/Hiera%20Tests/badge.svg)](https://github.com/lyraproj/hiera/actions)

## Introduction

Hiera is a flexible, powerful tool for resolving values for variable lookups, which was first popularised by its use in [Puppet](https://puppet.com/docs/puppet/5.5/hiera.html).

This module is a "clean-room" Go implementation of the Hiera framework, suitable for use as a library from other tools.

## Details

Hiera uses the concept of "managing by exception": you design a *hierarchy* of data sources, with the most specific source at the top and  least-specific defaults at the bottom. Hiera searches for keys starting at the top, allowing more-specific sources to override defaults. Sources are usually YAML files stored on the filesystem, and layers usually use variable interpolation to find the right file, allowing the context of the lookup to pick the right file.

## How to run it

### Standalone execution

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

## Extending Hiera

When Hiera performs a lookup it uses a lookup function. Unless the function embedded in the hiera binary, it will
make an attempt to load a RESTful Plugin that provides the needed function. Such plugins are enabled by using the API
provided by the [hierasdk library](https://github.com/lyraproj/hierasdk).

#### How Hiera finds its plugins
The resolution of a plugin can be controlled using a "plugindir" key in the Hiera configuration file. As with "datadir",
the "plugindir" can be specified both in the defaults hierarchy or explicitly in a specific hierarchy. Unless specified,
the "plugindir" is assumed to be the directory "plugin" present in the same directory as the configuration file.

In addition to "plugindir", a hierarchy may also specify a "pluginfile". Unless specified, the "pluginfile" is assumed
to be equal to the name of the lookup function (with the extension ".exe" in case of Windows).

## Environment Variables

The following environment variables can be set as an alternative to CLI options.

* `HIERA_CONFIGFILE` - `--config`

Values passed as CLI options will take precendence over the environment variables.

The following environment variables can be set as an alternative to setting values in the `defaults` hash.

* `HIERA_DATADIR` - `datadir`
* `HIERA_PLUGINDIR` - `plugindir`

Values set in `hiera.yaml` will take precedence over the environment variables.

### Containerized extension

In order to include an extension in a Hiera Docker image you need to:

1. Copy the source (or clone the git repository) of the desired extensions into the hiera plugin directory (don't worry,
   this directory will be ignored by git when doing commits to hiera).
2. For each extension, add a line like the following line to the Hiera Dockerfile below the comment
   `# Add plugin builds here`:
    ```
    RUN (cd plugin/hiera_terraform && go build -o ../terraform_backend)
    ```
3. Run the docker build.

### Useful extensions

* [Azure Key Vault lookup_key](https://github.com/lyraproj/hiera_azure). Allows you to lookup single values stored as
 secrets from the Azure Key Vault.
* [Terraform Backend data_hash](https://github.com/lyraproj/hiera_terrform). Allows hiera to query data from a Terraform
 backend. 

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
* [x] pluggable back ends
* [x] `explain` functionality to show traversal
* [x] containerized REST-based microservice
* [x] JSON and YAML schema for the hiera.yaml config file (see schema/hiera_v5.yaml)
