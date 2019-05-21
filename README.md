# Hiera lookup framework

## Introduction

Hiera is a flexible, powerful tool for resolving values for variable lookups, which was first popularised by its use in [Puppet](https://puppet.com/docs/puppet/5.5/hiera.html).

This module is a "clean-room" Go implementation of the Hiera framework, suitable for use as a library from other tools.

## How to use

#### Install the module

To install the module under $GOPATH/src:

    go get github.com/lyraproj/hiera

#### Install the lookup binary

Install the lookup binary under $GOPATH/bin:

    go install github.com/lyraproj/hiera/lookup

#### Run the binary

    lookup --help

## Details

Hiera uses the concept of "managing by exception": you design a *hierarchy* of data sources, with the most specific source at the top and  least-specific defaults at the bottom. Hiera searches for keys starting at the top, allowing more-specific sources to override defaults. Sources are usually YAML files stored on the filesystem, and layers usually use variable interpolation to find the right file, allowing the context of the lookup to pick the right file.

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
* [ ] layered hierarchy (global, environment, module)
