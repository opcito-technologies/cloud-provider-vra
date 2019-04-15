
package util

import (
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// indicates volume is attached to a different node
// than we expected.
type DanglingAttachError struct {
	msg         string
	CurrentNode k8stypes.NodeName
	DevicePath  string
}

func (err *DanglingAttachError) Error() string {
	return err.msg
}

// create a new dangling error
func NewDanglingError(msg string, node k8stypes.NodeName, devicePath string) error {
	return &DanglingAttachError{
		msg:         msg,
		CurrentNode: node,
		DevicePath:  devicePath,
	}
}
