# Hiera lookup framework

This module is a Go implementation of the Hiera 5 framework for Puppet. It will currently not accept Hiera 3 and it
cannot handle legacy Hiera 3 backends.

## Implementation status

* [x] lookup function
* [x] lookup context
* [x] dotted keys (dig functionality)
* [x] interpolation using scope function
* [x] interpolation using lookup/hiera function
* [x] interpolation using alias function
* [x] interpolation using literal function
* [ ] configuration
* [ ] layered hierarchy
* [ ] merge strategies
* [ ] YAML configuration
* [x] YAML data
* [ ] JSON data

