//go:build !coreless

package frame

import (
	"context"
	"errors"
	"fmt"

	client "github.com/EmptyDea-Team/EmptyDea-core-client"
	core_server "github.com/EmptyDea-Team/EmptyDea-core/frame/EmptyDeaCore/server"
	"github.com/google/uuid"
)

func (f *Frame) initClient() error {
	if f.client != nil {
		return nil
	}
	if !f.config.Embedded {
		return fmt.Errorf("Frame.initClient: nil client")
	}

	address := "fatalder-" + uuid.NewString()
	listener, err := core_server.Listen("shmipc", address, nil)
	if err != nil {
		return fmt.Errorf("Frame.initClient: listen embedded core: %w", err)
	}

	coreClient, err := client.DialContext(context.Background(), "shmipc", address)
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("Frame.initClient: dial embedded core: %w", err)
	}

	f.client = coreClient
	f.closer = func() error {
		return errors.Join(coreClient.Close(), listener.Close())
	}
	return nil
}
