//go:build coreless

package frame

import "fmt"

func (f *Frame) initClient() error {
	if f.client == nil {
		return fmt.Errorf("Frame.initClient: nil client")
	}
	return nil
}
