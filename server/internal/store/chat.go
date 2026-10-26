package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mini-wechat/server/internal/models"
)

// nowMs 返回 Unix 毫秒时间戳，全库时间字段统一使用该格式。
func nowMs() int64 { return time.Now().UnixMilli() }

// messageSelect 把消息与发送者一次 JOIN 出来，避免逐条查询发送人。
const messageSelect = `SELECT m.id, m.conversation_id, m.sender_id, m.type, m.content, m.created_at,
	u.id, u.username, u.nickname, u.avatar, u.signature
	FROM messages m JOIN users u ON u.id = m.sender_id`

// scanMessage 扫描消息及其发送者；没有行时返回 sql.ErrNoRows。
func scanMessage(row scanner) (models.Message, error) {
	var m models.Message
	var u models.User
	err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Type, &m.Content, &m.CreatedAt,
		&u.ID, &u.Username, &u.Nickname, &u.Avatar, &u.Signature)
	if err != nil {
		return m, err
	}
	m.Sender = &u
	return m, nil
}

// conversationMembers 按加入顺序返回会话成员。
func (s *Store) conversationMembers(convID int64) ([]models.User, error) {
	rows, err := s.db.Query(
		`SELECT u.id, u.username, u.nickname, u.avatar, u.signature
		 FROM conversation_members cm JOIN users u ON u.id = cm.user_id
		 WHERE cm.conversation_id = ? ORDER BY cm.joined_at, u.id`, convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []models.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		members = append(members, u)
	}
	return members, rows.Err()
}

// ListConversations 返回用户的会话列表，按最后一条消息时间倒序。
// 未读数在 SQL 中统计，并排除自己发出的消息。
func (s *Store) ListConversations(userID int64) ([]models.Conversation, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.type, c.name, c.owner_id, c.created_at,
		       m.id, m.sender_id, m.type, m.content, m.created_at,
		       su.id, su.username, su.nickname, su.avatar, su.signature,
		       (SELECT COUNT(*) FROM messages mm
		         WHERE mm.conversation_id = c.id AND mm.id > cm.last_read_message_id AND mm.sender_id <> ?) AS unread,
		       cm.last_read_message_id
		FROM conversations c
		JOIN conversation_members cm ON cm.conversation_id = c.id AND cm.user_id = ?
		LEFT JOIN messages m ON m.id = (SELECT MAX(id) FROM messages WHERE conversation_id = c.id)
		LEFT JOIN users su ON su.id = m.sender_id
		ORDER BY COALESCE(m.created_at, c.created_at) DESC`, userID, userID)
	if err != nil {
		return nil, err
	}

	conversations := []models.Conversation{}
	for rows.Next() {
		var cv models.Conversation
		var lastID, lastSender, lastCreated sql.NullInt64
		var lastType, lastContent sql.NullString
		var senderID sql.NullInt64
		var senderUsername, senderNickname, senderAvatar, senderSignature sql.NullString

		err := rows.Scan(
			&cv.ID, &cv.Type, &cv.Name, &cv.OwnerID, &cv.CreatedAt,
			&lastID, &lastSender, &lastType, &lastContent, &lastCreated,
			&senderID, &senderUsername, &senderNickname, &senderAvatar, &senderSignature,
			&cv.Unread, &cv.LastReadMessageID,
		)
		if err != nil {
			rows.Close()
			return nil, err
		}
		// LEFT JOIN 可能没有消息行，此时 lastID 为 NULL。
		if lastID.Valid {
			msg := &models.Message{
				ID:             lastID.Int64,
				ConversationID: cv.ID,
				SenderID:       lastSender.Int64,
				Type:           lastType.String,
				Content:        lastContent.String,
				CreatedAt:      lastCreated.Int64,
			}
			if senderID.Valid {
				msg.Sender = &models.User{
					ID: senderID.Int64, Username: senderUsername.String, Nickname: senderNickname.String,
					Avatar: senderAvatar.String, Signature: senderSignature.String,
				}
			}
			cv.LastMessage = msg
		}
		conversations = append(conversations, cv)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	// 连接池只有一条连接，必须先关闭结果集再执行后续的成员查询。
	rows.Close()

	for i := range conversations {
		members, err := s.conversationMembers(conversations[i].ID)
		if err != nil {
			return nil, err
		}
		conversations[i].Members = members
	}
	return conversations, nil
}

// ConversationForUser 返回单个会话详情；非成员查询时返回 ErrNotFound。
func (s *Store) ConversationForUser(convID, userID int64) (models.Conversation, error) {
	var cv models.Conversation
	var lastRead int64
	err := s.db.QueryRow(`
		SELECT c.id, c.type, c.name, c.owner_id, c.created_at, cm.last_read_message_id
		FROM conversations c
		JOIN conversation_members cm ON cm.conversation_id = c.id AND cm.user_id = ?
		WHERE c.id = ?`, userID, convID).Scan(
		&cv.ID, &cv.Type, &cv.Name, &cv.OwnerID, &cv.CreatedAt, &lastRead,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return cv, ErrNotFound
	}
	if err != nil {
		return cv, err
	}
	cv.LastReadMessageID = lastRead

	members, err := s.conversationMembers(convID)
	if err != nil {
		return cv, err
	}
	cv.Members = members

	msg, err := scanMessage(s.db.QueryRow(messageSelect+` WHERE m.conversation_id = ? ORDER BY m.id DESC LIMIT 1`, convID))
	if err == nil {
		cv.LastMessage = &msg
	} else if !errors.Is(err, sql.ErrNoRows) {
		return cv, err
	}

	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND id > ? AND sender_id <> ?`,
		convID, lastRead, userID,
	).Scan(&cv.Unread); err != nil {
		return cv, err
	}
	return cv, nil
}

// IsMember 判断用户是否为该会话成员，用于接口级权限校验。
func (s *Store) IsMember(convID, userID int64) (bool, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?`, convID, userID,
	).Scan(&n)
	return n > 0, err
}

// MemberIDs 返回会话全部成员 ID，用于消息与输入中提示的广播。
func (s *Store) MemberIDs(ctx context.Context, convID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id FROM conversation_members WHERE conversation_id = ?`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// FindDirect 查找两人之间已存在的单聊会话。
func (s *Store) FindDirect(a, b int64) (int64, error) {
	var id int64
	err := s.db.QueryRow(`
		SELECT c.id FROM conversations c
		JOIN conversation_members x ON x.conversation_id = c.id AND x.user_id = ?
		JOIN conversation_members y ON y.conversation_id = c.id AND y.user_id = ?
		WHERE c.type = 'direct' ORDER BY c.id LIMIT 1`, a, b).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

// CreateConversation 在单个事务内写入会话与成员关系。
func (s *Store) CreateConversation(kind, name string, ownerID int64, memberIDs []int64) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	// 提交成功后 Rollback 是空操作，因此可以无条件 defer。
	defer tx.Rollback()

	now := nowMs()
	res, err := tx.Exec(
		`INSERT INTO conversations (type, name, owner_id, created_at) VALUES (?, ?, ?, ?)`,
		kind, name, ownerID, now,
	)
	if err != nil {
		return 0, err
	}
	convID, _ := res.LastInsertId()
	for _, uid := range memberIDs {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO conversation_members (conversation_id, user_id, joined_at) VALUES (?, ?, ?)`,
			convID, uid, now,
		); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return convID, nil
}

// ListMessages 取 limit 条消息；beforeID > 0 时向前翻页。
// SQL 按 id 倒序取最近若干条，返回前再反转成正序，便于前端直接渲染。
func (s *Store) ListMessages(convID, beforeID int64, limit int) ([]models.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := messageSelect + ` WHERE m.conversation_id = ?`
	args := []any{convID}
	if beforeID > 0 {
		query += ` AND m.id < ?`
		args = append(args, beforeID)
	}
	query += ` ORDER BY m.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// CreateMessage 插入消息。发送者信息由调用方传入，省去一次回查。
func (s *Store) CreateMessage(convID, senderID int64, msgType, content string, sender models.User) (models.Message, error) {
	now := nowMs()
	res, err := s.db.Exec(
		`INSERT INTO messages (conversation_id, sender_id, type, content, created_at) VALUES (?, ?, ?, ?, ?)`,
		convID, senderID, msgType, content, now,
	)
	if err != nil {
		return models.Message{}, err
	}
	id, _ := res.LastInsertId()
	return models.Message{
		ID: id, ConversationID: convID, SenderID: senderID, Type: msgType,
		Content: content, CreatedAt: now, Sender: &sender,
	}, nil
}

// MarkRead 将会话已读位点向前推进；messageID <= 0 表示标记到最新一条。
func (s *Store) MarkRead(convID, userID, messageID int64) error {
	if messageID <= 0 {
		if err := s.db.QueryRow(
			`SELECT COALESCE(MAX(id), 0) FROM messages WHERE conversation_id = ?`, convID,
		).Scan(&messageID); err != nil {
			return err
		}
	}
	// last_read_message_id < ? 保证位点只会前进，防止并发请求把已读状态回退。
	_, err := s.db.Exec(
		`UPDATE conversation_members SET last_read_message_id = ?
		 WHERE conversation_id = ? AND user_id = ? AND last_read_message_id < ?`,
		messageID, convID, userID, messageID,
	)
	return err
}
