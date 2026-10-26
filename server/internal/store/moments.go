package store

import (
	"database/sql"
	"encoding/json"
	"errors"

	"mini-wechat/server/internal/models"
)

// momentSelect 动态与作者一次 JOIN 取出。
const momentSelect = `SELECT mo.id, mo.content, mo.images, mo.created_at,
	u.id, u.username, u.nickname, u.avatar, u.signature
	FROM moments mo JOIN users u ON u.id = mo.user_id`

// scanMoment 扫描动态主体；图片以 JSON 数组文本存库，点赞和评论由后续查询补全。
func scanMoment(row scanner) (models.Moment, error) {
	var m models.Moment
	var imagesJSON string
	err := row.Scan(
		&m.ID, &m.Content, &imagesJSON, &m.CreatedAt,
		&m.User.ID, &m.User.Username, &m.User.Nickname, &m.User.Avatar, &m.User.Signature,
	)
	if err != nil {
		return m, err
	}
	// 预置为空切片，序列化给前端时才是 [] 而不是 null。
	m.Images = []string{}
	m.Likes = []models.User{}
	m.Comments = []models.MomentComment{}
	_ = json.Unmarshal([]byte(imagesJSON), &m.Images)
	return m, nil
}

// MomentLikes 返回动态的点赞用户列表，按点赞时间升序。
func (s *Store) MomentLikes(momentID int64) ([]models.User, error) {
	rows, err := s.db.Query(
		`SELECT u.id, u.username, u.nickname, u.avatar, u.signature
		 FROM moment_likes ml JOIN users u ON u.id = ml.user_id
		 WHERE ml.moment_id = ? ORDER BY ml.created_at`, momentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	likes := []models.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		likes = append(likes, u)
	}
	return likes, rows.Err()
}

// momentComments 返回动态的评论列表，按 ID 升序即时间正序。
func (s *Store) momentComments(momentID int64) ([]models.MomentComment, error) {
	rows, err := s.db.Query(
		`SELECT c.id, c.content, c.created_at, u.id, u.username, u.nickname, u.avatar, u.signature
		 FROM moment_comments c JOIN users u ON u.id = c.user_id
		 WHERE c.moment_id = ? ORDER BY c.id`, momentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []models.MomentComment{}
	for rows.Next() {
		var cm models.MomentComment
		err := rows.Scan(
			&cm.ID, &cm.Content, &cm.CreatedAt,
			&cm.User.ID, &cm.User.Username, &cm.User.Nickname, &cm.User.Avatar, &cm.User.Signature,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, cm)
	}
	return comments, rows.Err()
}

// enrichMoment 补全点赞与评论，并计算查看者是否已点赞。
func (s *Store) enrichMoment(m *models.Moment, viewerID int64) error {
	likes, err := s.MomentLikes(m.ID)
	if err != nil {
		return err
	}
	m.Likes = likes
	for _, u := range likes {
		if u.ID == viewerID {
			m.LikedByMe = true
		}
	}
	comments, err := s.momentComments(m.ID)
	if err != nil {
		return err
	}
	m.Comments = comments
	return nil
}

// ListMoments 返回最新 limit 条动态。
func (s *Store) ListMoments(viewerID int64, limit int) ([]models.Moment, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(momentSelect+` ORDER BY mo.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}

	moments := []models.Moment{}
	for rows.Next() {
		m, err := scanMoment(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		moments = append(moments, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	// 数据库只有一条连接，必须关闭结果集后才能继续查询点赞和评论。
	rows.Close()

	for i := range moments {
		if err := s.enrichMoment(&moments[i], viewerID); err != nil {
			return nil, err
		}
	}
	return moments, nil
}

// MomentByID 返回单条动态详情。
func (s *Store) MomentByID(viewerID, momentID int64) (models.Moment, error) {
	m, err := scanMoment(s.db.QueryRow(momentSelect+` WHERE mo.id = ?`, momentID))
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, err
	}
	if err := s.enrichMoment(&m, viewerID); err != nil {
		return m, err
	}
	return m, nil
}

// CreateMoment 发布动态，图片列表以 JSON 文本存库。
func (s *Store) CreateMoment(userID int64, content string, images []string) (int64, error) {
	encoded, _ := json.Marshal(images)
	res, err := s.db.Exec(
		`INSERT INTO moments (user_id, content, images, created_at) VALUES (?, ?, ?, ?)`,
		userID, content, string(encoded), nowMs(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ToggleLike 点赞/取消点赞，返回操作后的点赞状态。
func (s *Store) ToggleLike(momentID, userID int64) (bool, error) {
	var exists int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM moment_likes WHERE moment_id = ? AND user_id = ?`, momentID, userID,
	).Scan(&exists); err != nil {
		return false, err
	}
	if exists > 0 {
		_, err := s.db.Exec(`DELETE FROM moment_likes WHERE moment_id = ? AND user_id = ?`, momentID, userID)
		return false, err
	}
	_, err := s.db.Exec(
		`INSERT INTO moment_likes (moment_id, user_id, created_at) VALUES (?, ?, ?)`,
		momentID, userID, nowMs(),
	)
	return true, err
}

// AddComment 添加评论并返回带作者信息的完整对象。
func (s *Store) AddComment(momentID, userID int64, content string) (models.MomentComment, error) {
	now := nowMs()
	res, err := s.db.Exec(
		`INSERT INTO moment_comments (moment_id, user_id, content, created_at) VALUES (?, ?, ?, ?)`,
		momentID, userID, content, now,
	)
	if err != nil {
		return models.MomentComment{}, err
	}
	id, _ := res.LastInsertId()
	u, err := s.UserByID(userID)
	if err != nil {
		return models.MomentComment{}, err
	}
	return models.MomentComment{ID: id, User: u, Content: content, CreatedAt: now}, nil
}
