module github.com/carapace-sh/carapace-shlex/cmd

go 1.22.0

require (
	github.com/carapace-sh/carapace v1.0.0
	github.com/carapace-sh/carapace-bridge v0.2.16-0.20240311173237-760172108463
	github.com/carapace-sh/carapace-shlex v1.0.0
	github.com/spf13/cobra v1.8.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/carapace-sh/carapace-shlex => ../
