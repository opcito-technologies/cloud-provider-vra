
package volume

const (
	// Name of a volume in external cloud that is being provisioned and thus
	// should be ignored by rest of Kubernetes.
	ProvisionedVolumeName = "placeholder-for-provisioning"
)

// NewDeletedVolumeInUseError returns a new instance of DeletedVolumeInUseError
// error.
func NewDeletedVolumeInUseError(message string) error {
	return deletedVolumeInUseError(message)
}

type deletedVolumeInUseError string

var _ error = deletedVolumeInUseError("")

// IsDeletedVolumeInUse returns true if an error returned from Delete() is
// deletedVolumeInUseError
func IsDeletedVolumeInUse(err error) bool {
	switch err.(type) {
	case deletedVolumeInUseError:
		return true
	default:
		return false
	}
}

func (err deletedVolumeInUseError) Error() string {
	return string(err)
}
