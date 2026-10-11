package snowflake_storage

import (
	"context"
	"strings"

	sql_based_storage "github.com/storage-lock/go-sql-based-storage"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	"time"
)

// SnowflakeStorage 基于 Snowflake 的存储引擎实现
type SnowflakeStorage struct {
	*sql_based_storage.SqlBasedStorage
}

func NewSnowflakeStorage(ctx context.Context, options *sql_based_storage.SqlBasedStorageOptions) (*SnowflakeStorage, error) {
	// 如果调用方没有显式指定方言（还是默认的 Sql92Provider），换成 Snowflake 自己的方言提供者：
	// Sql92Provider 的裸 INSERT CreateWithVersionSql 在 Snowflake 上不能保证 CAS 互斥（见
	// SnowflakeSqlProvider 注释），NowTimestampSql 也是 MySQL 专有语法，必须替换掉。
	if _, ok := options.SqlProvider.(*sql_based_storage.Sql92Provider); ok || options.SqlProvider == nil {
		options.SetSqlProvider(NewSnowflakeSqlProvider())
	}
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

// CreateWithVersion 防御性兜底：MERGE 语句本身已经解决了 CAS 互斥（见 SnowflakeSqlProvider），
// 这里只是防止 Snowflake 并发事务冲突时抛出的错误被当成不可重试的硬错误往上传。
// 注意：未接入真实 Snowflake 账号验证，错误消息按官方文档做的最佳努力匹配。
func (x *SnowflakeStorage) CreateWithVersion(ctx context.Context, lockId string, version storage.Version, lockInformation *storage.LockInformation) error {
	err := x.SqlBasedStorage.CreateWithVersion(ctx, lockId, version, lockInformation)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "oncurrent") || strings.Contains(msg, "duplicate") {
			return storage_lock.ErrLockAlreadyExists
		}
	}
	return err
}

const StorageName = "snowflake-storage"

func (x *SnowflakeStorage) GetName() string {
	return StorageName
}

func (x *SnowflakeStorage) Capabilities() []storage.StorageCapability {
	return []storage.StorageCapability{
		storage.CapabilityCAS,
		storage.CapabilityReliableTime,
		storage.CapabilityWatch,
	}
}

func (x *SnowflakeStorage) Watch(ctx context.Context, lockId string) (<-chan *storage.LockInformation, error) {
	return storage.NewPollingLockWatcher(x, 500*time.Millisecond).Watch(ctx, lockId)
}
