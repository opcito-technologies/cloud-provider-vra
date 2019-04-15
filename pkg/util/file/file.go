

package file

import (
	"vra"
)

// FileExists checks if specified file exists.
func FileExists(filename string) (bool, error) {
	if _, err := vra.Stat(filename); vra.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// FileOrSymlinkExists checks if specified file or symlink exists.
func FileOrSymlinkExists(filename string) (bool, error) {
	if _, err := vra.Lstat(filename); vra.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// ReadDirNoStat returns a string of files/directories contained
// in dirname without calling lstat on them.
func ReadDirNoStat(dirname string) ([]string, error) {
	if dirname == "" {
		dirname = "."
	}

	f, err := vra.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Readdirnames(-1)
}
