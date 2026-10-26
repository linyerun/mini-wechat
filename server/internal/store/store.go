// Package store 基于 SQLite 实现数据持久化：建表迁移、演示数据种子与各类查询。
// 驱动为纯 Go 的 modernc.org/sqlite，因此无需 CGO 即可静态编译。
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"mini-wechat/server/internal/models"
)

// 哨兵错误：便于上层用 errors.Is 区分“不存在”和“唯一约束冲突”。
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// Store 包装数据库连接池。
type Store struct {
	db *sql.DB
}

// Open 打开（必要时创建）数据库文件，完成迁移并写入演示数据。
// DSN 中的三个 pragma 分别用于：写冲突等待、WAL 日志模式、开启外键约束。
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite 的写入本身就是串行的，限制为单连接可以彻底避开 "database is locked"。
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.seed(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error { return s.db.Close() }

// schema 全部使用 IF NOT EXISTS，可以重复执行；时间字段统一存 Unix 毫秒。
const schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	nickname TEXT NOT NULL,
	avatar TEXT NOT NULL DEFAULT '',
	signature TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS conversations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL,
	name TEXT NOT NULL DEFAULT '',
	owner_id INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS conversation_members (
	conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	last_read_message_id INTEGER NOT NULL DEFAULT 0,
	joined_at INTEGER NOT NULL,
	PRIMARY KEY (conversation_id, user_id)
);

CREATE TABLE IF NOT EXISTS messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
	sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	type TEXT NOT NULL DEFAULT 'text',
	content TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, id);

CREATE TABLE IF NOT EXISTS moments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	content TEXT NOT NULL DEFAULT '',
	images TEXT NOT NULL DEFAULT '[]',
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS moment_likes (
	moment_id INTEGER NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (moment_id, user_id)
);

CREATE TABLE IF NOT EXISTS moment_comments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	moment_id INTEGER NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	content TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
`

// migrate 执行建表语句。
func (s *Store) migrate() error {
	_, err := s.db.Exec(schema)
	return err
}

// scanner 抽象 QueryRow 与 Query 的共用扫描方法。
type scanner interface {
	Scan(dest ...any) error
}

// scanUser 按固定列顺序扫描用户，注意不包含 password_hash。
func scanUser(row scanner) (models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Nickname, &u.Avatar, &u.Signature)
	return u, err
}

// UserByUsername 返回用户及其密码哈希，仅供登录校验使用。
func (s *Store) UserByUsername(username string) (models.User, string, error) {
	var u models.User
	var hash string
	err := s.db.QueryRow(
		`SELECT id, username, nickname, avatar, signature, password_hash FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Nickname, &u.Avatar, &u.Signature, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return u, "", ErrNotFound
	}
	return u, hash, err
}

// UserByID 按主键查询用户。
func (s *Store) UserByID(id int64) (models.User, error) {
	u, err := scanUser(s.db.QueryRow(
		`SELECT id, username, nickname, avatar, signature FROM users WHERE id = ?`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// ListUsers 返回通讯录列表，排除用户自己。
func (s *Store) ListUsers(excludeID int64) ([]models.User, error) {
	rows, err := s.db.Query(
		`SELECT id, username, nickname, avatar, signature FROM users WHERE id <> ? ORDER BY nickname`, excludeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// CreateUser 注册新用户。驱动没有导出错误类型，只能按 UNIQUE 报错文本判定重名。
func (s *Store) CreateUser(username, passwordHash, nickname string) (models.User, error) {
	now := nowMs()
	res, err := s.db.Exec(
		`INSERT INTO users (username, password_hash, nickname, created_at) VALUES (?, ?, ?, ?)`,
		username, passwordHash, nickname, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return models.User{}, ErrConflict
		}
		return models.User{}, err
	}
	id, _ := res.LastInsertId()
	return models.User{ID: id, Username: username, Nickname: nickname}, nil
}

// UpdateUser 局部更新资料，指针为 nil 表示该字段保持原值。
func (s *Store) UpdateUser(id int64, nickname, signature, avatar *string) (models.User, error) {
	if nickname != nil {
		if _, err := s.db.Exec(`UPDATE users SET nickname = ? WHERE id = ?`, *nickname, id); err != nil {
			return models.User{}, err
		}
	}
	if signature != nil {
		if _, err := s.db.Exec(`UPDATE users SET signature = ? WHERE id = ?`, *signature, id); err != nil {
			return models.User{}, err
		}
	}
	if avatar != nil {
		if _, err := s.db.Exec(`UPDATE users SET avatar = ? WHERE id = ?`, *avatar, id); err != nil {
			return models.User{}, err
		}
	}
	return s.UserByID(id)
}

// UsersExist 判断给定 ID 是否全部存在。占位符数量按入参长度拼接，
// 取值仍走参数绑定，因此不存在 SQL 注入风险。
func (s *Store) UsersExist(ids []int64) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE id IN (`+placeholders+`)`, args...).Scan(&n)
	return n == len(ids), err
}
