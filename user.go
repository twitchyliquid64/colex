package colex

import (
	"syscall"
)

const (
	// NamespaceUser indicates a new user namespace should be created for the process.
	NamespaceUser = syscall.CLONE_NEWUSER
)

// MapUser declares a mapping between the UID of the executing usernamespace and the to-be-created one.
func MapUser(hostID, containedID int) syscall.SysProcIDMap {
	return syscall.SysProcIDMap{
		ContainerID: containedID,
		HostID:      hostID,
		Size:        1,
	}
}

// MapGroup declares a mapping between the GID of the executing usernamespace and the to-be-created one.
func MapGroup(hostID, containedID int) syscall.SysProcIDMap {
	return syscall.SysProcIDMap{
		ContainerID: containedID,
		HostID:      hostID,
		Size:        1,
	}
}
