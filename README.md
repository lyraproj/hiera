# Hiera lookup framework

## Introduction

Hiera is a flexible, powerful tool for resolving values for variable lookups, which was first popularised by its use in [Puppet](https://puppet.com/docs/puppet/5.5/hiera.html).

This module is a "clean-room" Go implementation of the Hiera framework, suitable for use as a library from other tools.

## Details

Hiera uses the concept of "managing by exception": you design a *hierarchy* of data sources, with the most specific source at the top and  least-specific defaults at the bottom. Hiera searches for keys starting at the top, allowing more-specific sources to override defaults. Sources are usually YAML files stored on the filesystem, and layers usually use variable interpolation to find the right file, allowing the context of the lookup to pick the right file.

## Implementation status

* [x] lookup function
* [x] lookup context
* [x] dotted keys (dig functionality)
* [x] interpolation using scope function
* [x] interpolation using lookup/hiera function
* [x] interpolation using alias function
* [x] interpolation using literal function
* [x] configuration
* [x] merge strategies (first, unique, hash, deep)
* [x] YAML configuration
* [x] YAML data
* [x] JSON data
* [x] lookup options stored adjacent to data
* [x] convert_to type coersions
* [x] Sensitive data
* [ ] configurable deep merge
* [ ] layered hierarchy (global, environment, module)
