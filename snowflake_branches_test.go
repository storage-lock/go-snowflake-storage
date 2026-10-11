package snowflake_storage

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	sql_based_storage "github.com/storage-lock/go-sql-based-storage"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	"github.com/stretchr/testify/assert"
)

type stubProvider struct{ *sql_based_storage.Sql92Provider }

func (stubProvider) CreateTableSql(context.Context, string) (string, []any) {
	return "SELECT 1", nil
}

func TestSnowflakeSqlProvider(t *testing.T) {
	provider := NewSnowflakeSqlProvider()
	sqlText, args := provider.NowTimestampSql(context.Background(), "locks")
	assert.Contains(t, sqlText, "DATE_PART")
	assert.Nil(t, args)

	info := &storage.LockInformation{OwnerId: "owner"}
	mergeSQL, params := provider.CreateWithVersionSql(context.Background(), "locks", "lock-1", 3, info)
	assert.Contains(t, mergeSQL, "MERGE INTO locks")
	assert.Equal(t, []any{"lock-1", "lock-1", "owner", storage.Version(3), info.ToJsonString()}, params)
}

func TestNewSnowflakeStorage_NilOptions(t *testing.T) {
	assert.Panics(t, func() { _, _ = NewSnowflakeStorage(context.Background(), nil) })
}

func TestNewSnowflakeStorage_InvalidOptions(t *testing.T) {
	_, err := NewSnowflakeStorage(context.Background(), &sql_based_storage.SqlBasedStorageOptions{})
	assert.Error(t, err)
}

func TestNewSnowflakeStorage_InitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	mock.ExpectExec("CREATE TABLE").WillReturnError(errors.New("init failed"))
	_, err = NewSnowflakeStorage(context.Background(), sql_based_storage.NewSqlBasedStorageOptions().
		SetConnectionManager(storage.NewFixedSqlDBConnectionManager(db)))
	assert.EqualError(t, err, "init failed")
}

func TestNewSnowflakeStorage_CustomProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
	s, err := NewSnowflakeStorage(context.Background(), sql_based_storage.NewSqlBasedStorageOptions().
		SetConnectionManager(storage.NewFixedSqlDBConnectionManager(db)).
		SetSqlProvider(stubProvider{Sql92Provider: sql_based_storage.NewSql92Provider()}))
	assert.NoError(t, err)
	assert.NotNil(t, s)
}

func TestSnowflakeStorage_CreateWithVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	s, err := NewSnowflakeStorage(context.Background(), sql_based_storage.NewSqlBasedStorageOptions().
		SetConnectionManager(storage.NewFixedSqlDBConnectionManager(db)))
	assert.NoError(t, err)
	info := &storage.LockInformation{OwnerId: "owner"}

	mock.ExpectExec("MERGE INTO").WillReturnError(errors.New("concurrent modification"))
	assert.ErrorIs(t, s.CreateWithVersion(context.Background(), "lock", 1, info), storage_lock.ErrLockAlreadyExists)
	mock.ExpectExec("MERGE INTO").WillReturnError(errors.New("duplicate row"))
	assert.ErrorIs(t, s.CreateWithVersion(context.Background(), "lock", 1, info), storage_lock.ErrLockAlreadyExists)
	mock.ExpectExec("MERGE INTO").WillReturnError(errors.New("other"))
	assert.EqualError(t, s.CreateWithVersion(context.Background(), "lock", 1, info), "other")
	mock.ExpectExec("MERGE INTO").WillReturnResult(sqlmock.NewResult(1, 1))
	assert.NoError(t, s.CreateWithVersion(context.Background(), "lock", 1, info))
}

func TestSnowflakeStorage_Watch_EmptyLockID(t *testing.T) {
	_, err := (&SnowflakeStorage{}).Watch(context.Background(), "")
	assert.Error(t, err)
}
