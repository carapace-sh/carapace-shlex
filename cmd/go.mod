module github.com/carapace-sh/carapace-shlex/v2/cmd

go 1.24.0

require (
	github.com/carapace-sh/carapace v1.16.4-0.20260930123217-cb6bcc38658f
	github.com/carapace-sh/carapace-bridge v1.6.4
	github.com/carapace-sh/carapace-shlex/v2 v2.0.0-20260930122814-3264fae00e3d
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/carapace-sh/carapace-shlex v1.1.2-0.20260701213017-3985f5788c03 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

replace github.com/carapace-sh/carapace-shlex/v2 => ../
