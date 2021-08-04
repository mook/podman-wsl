package winapi

import "fmt"

func Debug(message string) error {
	err := OutputDebugstring(message)
	if err != nil {
		return err
	}
	return nil
}

func Debugf(message string, args ...interface{}) error {
	err := OutputDebugstring(fmt.Sprintf(message, args...))
	if err != nil {
		return err
	}
	return nil
}
