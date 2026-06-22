module healthwatch

go 1.22

require (
	github.com/cucumber/godog v0.15.1
	github.com/goccy/go-yaml v1.19.2
)

require (
	github.com/cucumber/gherkin/go/v26 v26.2.0 // indirect
	github.com/cucumber/messages/go/v21 v21.0.1 // indirect
	github.com/gofrs/uuid v4.3.1+incompatible // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.4 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
)

// go-yaml/yaml.v3 and check.v1 are pulled in transitively by godog via their
// gopkg.in vanity import paths. We point them at their canonical GitHub
// mirrors so the whole module graph resolves from github.com only - no
// dependency on the gopkg.in redirect service at all.
replace gopkg.in/check.v1 => github.com/go-check/check v0.0.0-20201130134442-10cb98267c6c

replace gopkg.in/yaml.v3 => github.com/go-yaml/yaml v0.0.0-20250401170010-944c86a7d293
