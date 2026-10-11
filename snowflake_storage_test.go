package snowflake_storage

import (
	"context"
	"testing"

	"github.com/storage-lock/go-storage"
	"github.com/stretchr/testify/assert"
)

func TestSnowflakeStorage_GetName(t *testing.T) {
	s := &SnowflakeStorage{}
	assert.Equal(t, StorageName, s.GetName())
}

func TestSnowflakeStorage_Capabilities(t *testing.T) {
	s := &SnowflakeStorage{}
	caps := s.Capabilities()
	assert.Contains(t, caps, storage.CapabilityCAS)
	assert.Contains(t, caps, storage.CapabilityReliableTime)
}

func TestSnowflakeStorage_Close(t *testing.T) {
	s := &SnowflakeStorage{}
	err := s.Close(context.Background())
	assert.NoError(t, err)
}
