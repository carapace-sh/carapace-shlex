module github.com/carapace-sh/carapace-shlex/cmd

go 1.23.1

require (
	github.com/carapace-sh/carapace v1.8.6
	github.com/carapace-sh/carapace-bridge v1.4.1
	github.com/carapace-sh/carapace-shlex v1.0.1
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/carapace-sh/carapace-shlex => ../
