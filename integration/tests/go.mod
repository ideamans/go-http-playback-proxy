module integration-tests

go 1.22.0

toolchain go1.22.2

require (
	github.com/stretchr/testify v1.10.0
	go-http-playback-proxy v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go-http-playback-proxy => ../../
