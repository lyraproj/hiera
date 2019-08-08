## Schema for the hiera.yaml config

This directory contains the [json schema](https://json-schema.org/) for the hiera.yaml configuration. The schema is maintained
in yaml and the json version is generated from this directory like so:
```bash
go run ../yaml2json < hiera_v5.yaml > hiera_v5.json
```
