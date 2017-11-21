package colex

import "syscall"

const (
	// NamespaceDomains indicates a new UTS namespace should be created for the process.
	NamespaceDomains = syscall.CLONE_NEWUTS
)
