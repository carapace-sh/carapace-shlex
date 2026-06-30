package shlex

// OilFormat returns the Oil shell (OSH) lexical format.
// OSH is bash-compatible, so it uses the bash format directly.
// YSH string types (r'...', ”'...”') are deferred to Phase 4.
func OilFormat() Format { return bashFormat{} }
