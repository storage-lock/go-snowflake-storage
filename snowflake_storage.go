package snowflake_storage

import (
	"context"

	sql_based_storage "github.com/storage-lock/go-sql-based-storage"
	"github.com/storage-lock/go-storage"
)

// SnowflakeStorage 基于 Snowflake 的存储引擎实现
type SnowflakeStorage struct {
	*sql_based_storage.SqlBasedStorage
}

func NewSnowflakeStorage(ctx context.Context, options *sql_based_storage.SqlBasedStorageOptions) (*SnowflakeStorage, error) {
	baseStorage, err := sql_based_storage.NewSqlBasedStorage(options)
	if err != nil {
		return nil, err
	}
	s := &SnowflakeStorage{SqlBasedStorage: baseStorage}
	if err := s.Init(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

const StorageName = "snowflake-storage"

func (x *SnowflakeStorage) GetName() string {
	return StorageName
}

func (x *SnowflakeStorage) Capabilities() []storage.StorageCapability {
	return []storage.StorageCapability{
		storage.CapabilityCAS,
		storage.CapabilityReliableTime,
	}
}
