package cmd

import "os"

type NotFoundError struct {
	Err error
}

func (n NotFoundError) Error() string {
	return n.Error()
}

func isDirectory(path string) bool {
	// Get file information
	info, err := os.Stat(path)
	if err != nil {
		// Error occurred, path does not exist or cannot be accessed
		return false
	}

	// Check if the path is a directory
	return info.Mode().IsDir()
}
