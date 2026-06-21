//go:build !coreless

package frame

import (
	"fmt"

	core_direct_client "github.com/EmptyDea-Team/EmptyDea-core/frame/EmptyDeaCore/client"
)

func (f *Frame) initClient() error {
	if f.client != nil {
		return nil
	}
	if !f.config.Embedded {
		return fmt.Errorf("Frame.initClient: nil client")
	}

	f.client = core_direct_client.New(nil)
	f.closer = f.client.Close
	return nil
}
