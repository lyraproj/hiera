FROM golang:alpine as build_base

WORKDIR /go/src/github.com/lyraproj/hiera
COPY . .

# Since alpine doesn't have gcc
RUN apk add --no-cache gcc musl-dev openssl

RUN go install ./...


# Ensure that a plugin directory is present
RUN mkdir -p plugin

# Include external plugins. A prerequisite is that the plugin source is copied into the hiera/plugin directory
# prior to the docker build.

# Add plugin builds here...

# hiera_terraform plugin
# RUN (cd plugin/hiera_terraform && go build -o ../terraform_backend)

# hiera_azure plugin
# RUN (cd plugin/hiera_azure && go build -o ../azure_key_vault)

# Create a new minimalisic image that doesn't contain the build environment and
# copy the executable over
FROM alpine
COPY --from=build_base /go/bin/rest /bin/hieraserver
RUN mkdir -p /hiera/plugin
# COPY --from=build_base /go/src/github.com/lyraproj/hiera/plugin/* /hiera/plugin/

# Configurable values for runtime overrides
ENV port 8080
ENV loglevel error
ENV config /hiera/hiera.yaml

CMD /bin/hieraserver --port ${port} --loglevel ${loglevel} --config ${config}
