package colex

import "syscall"

const (
	// NamespaceNet indicates a new Network namespace should be created for the process.
	NamespaceNet = syscall.CLONE_NEWNET
)
