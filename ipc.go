package colex

import "syscall"

const (
	// NamespaceIPC indicates a new Inter Process Communication namespace should be created for the process.
	NamespaceIPC = syscall.CLONE_NEWIPC
)
