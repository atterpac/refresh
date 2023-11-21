//go:build !windows && !linux && !darwin

package engine

func init() {
	println("Unsupported OS detected. File watching will not work.")
	os.Exit(1)
}
