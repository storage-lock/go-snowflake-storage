package snowflake_storage

import (
	"context"
	"fmt"

	sql_based_storage "github.com/storage-lock/go-sql-based-storage"
	"github.com/storage-lock/go-storage"
)

// SnowflakeSqlProvider Snowflake 的 SQL 方言提供者
//
// 特别注意：Snowflake 的 PRIMARY KEY / UNIQUE 约束默认只是元数据声明，查询引擎并不会强制
// 执行——裸 INSERT 遇到重复的 lock_id 不会报错，而是直接插入第二行。这意味着继承
// Sql92Provider 的裸 INSERT CreateWithVersionSql 在 Snowflake 上完全不能保证 CAS 互斥，
// 不是"漏映射一个错误"能解决的，必须换成 MERGE INTO ... WHEN NOT MATCHED THEN INSERT，
// 让"是否插入"这件事本身由语句执行时条件判断，不依赖约束强制。
//
// 注意：未接入真实 Snowflake 账号/驱动验证，MERGE 语句与 RowsAffected 语义是按 Snowflake
// 官方文档（MERGE 返回受影响行数 = 插入数 + 更新数之和）做的最佳努力实现。
type SnowflakeSqlProvider struct {
	*sql_based_storage.Sql92Provider
}

func NewSnowflakeSqlProvider() *SnowflakeSqlProvider {
	return &SnowflakeSqlProvider{
		Sql92Provider: sql_based_storage.NewSql92Provider(),
	}
}

// NowTimestampSql Snowflake 获取当前时间的 Unix 时间戳
// 修复：原来直接继承 Sql92Provider 的 "SELECT UNIX_TIMESTAMP(NOW())"，Snowflake 没有这两个
// MySQL 专有函数，执行即报错，导致 GetTime 完全不可用。Snowflake 内置 DATE_PART('epoch_second', ...)
// 可以直接把时间戳换算为 Unix 秒数，无需手工日期算术。
func (x *SnowflakeSqlProvider) NowTimestampSql(ctx context.Context, tableFullName string) (string, []any) {
	return `SELECT DATE_PART('epoch_second', CURRENT_TIMESTAMP())::NUMBER(20,0)`, nil
}

// CreateWithVersionSql 用 MERGE 而不是裸 INSERT 实现 CAS 语义
// WHEN NOT MATCHED 时才 INSERT，已存在 lock_id 时 MERGE 不做任何事，RowsAffected 为 0，
// 交由基类 CreateWithVersion 的 "affected == 0 → ErrVersionMiss" 逻辑处理（core 视为可重试）。
func (x *SnowflakeSqlProvider) CreateWithVersionSql(ctx context.Context, tableFullName string, lockId string, version storage.Version, lockInformation *storage.LockInformation) (string, []any) {
	mergeSql := fmt.Sprintf(`MERGE INTO %s AS t USING (SELECT ? AS lock_id) AS s ON t.lock_id = s.lock_id
WHEN NOT MATCHED THEN INSERT (lock_id, owner_id, version, lock_information_json_string) VALUES (?, ?, ?, ?)`, tableFullName)
	return mergeSql, []any{lockId, lockId, lockInformation.OwnerId, version, lockInformation.ToJsonString()}
}
