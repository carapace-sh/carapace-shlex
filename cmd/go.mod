module github.com/carapace-sh/carapace-shlex/cmd

go 1.22.0
toolchain go1.23.6

require (
	github.com/carapace-sh/carapace v1.7.1
	github.com/carapace-sh/carapace-bridge v1.2.3
	github.com/carapace-sh/carapace-shlex v1.0.1
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/carapace-sh/carapace-shlex => ../
