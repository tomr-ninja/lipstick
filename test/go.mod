module github.com/tomr-ninja/lipstick/test

go 1.24.0

replace github.com/tomr-ninja/lipstick => ../

require (
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
	github.com/stretchr/testify v1.11.1
	github.com/tomr-ninja/lipstick v0.0.0-00010101000000-000000000000
	gopkg.in/hraban/opus.v2 v2.0.0-20230925203106-0188a62cb302
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-audio/riff v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
