package store

import (
	"time"

	"mini-wechat/server/internal/auth"
)

// seed 在全新数据库中写入演示用户、会话、消息与朋友圈，
// 让项目首次启动就有内容可看。
func (s *Store) seed() error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	// 只要已有用户就说明种子过了，避免重复写入。
	if count > 0 {
		return nil
	}

	// 演示账号共用同一密码，哈希只需计算一次。
	hash, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UnixMilli()
	ago := func(minutes int64) int64 { return now - minutes*60_000 }

	// 闭包内不便向上返回 error，用共享标记记录首个失败并短路后续写入。
	var seedErr error

	users := []struct{ username, nickname, signature string }{
		{"alice", "林小敏", "生活明朗，万物可爱"},
		{"bob", "陈杰", "在路上，也在生活里"},
		{"carol", "苏晴", "今天也要开心呀 ☀️"},
		{"dave", "王叔", "岁月静好"},
		{"emma", "周雨桐", "用镜头记录小确幸 📷"},
	}
	ids := map[string]int64{}
	for _, u := range users {
		res, err := tx.Exec(
			`INSERT INTO users (username, password_hash, nickname, signature, created_at) VALUES (?, ?, ?, ?, ?)`,
			u.username, hash, u.nickname, u.signature, ago(60*24*30),
		)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		ids[u.username] = id
	}

	createConv := func(kind, name string, ownerID int64, memberIDs []int64) int64 {
		if seedErr != nil {
			return 0
		}
		res, err := tx.Exec(
			`INSERT INTO conversations (type, name, owner_id, created_at) VALUES (?, ?, ?, ?)`,
			kind, name, ownerID, ago(60*24*7),
		)
		if err != nil {
			seedErr = err
			return 0
		}
		id, _ := res.LastInsertId()
		for _, uid := range memberIDs {
			if _, err := tx.Exec(
				`INSERT INTO conversation_members (conversation_id, user_id, joined_at) VALUES (?, ?, ?)`,
				id, uid, ago(60*24*7),
			); err != nil {
				seedErr = err
				return 0
			}
		}
		return id
	}

	addMsg := func(convID, senderID int64, content string, createdAt int64) int64 {
		if seedErr != nil {
			return 0
		}
		res, err := tx.Exec(
			`INSERT INTO messages (conversation_id, sender_id, type, content, created_at) VALUES (?, ?, 'text', ?, ?)`,
			convID, senderID, content, createdAt,
		)
		if err != nil {
			seedErr = err
			return 0
		}
		id, _ := res.LastInsertId()
		return id
	}

	setRead := func(convID, userID, messageID int64) {
		if seedErr != nil {
			return
		}
		if _, err := tx.Exec(
			`UPDATE conversation_members SET last_read_message_id = ? WHERE conversation_id = ? AND user_id = ?`,
			messageID, convID, userID,
		); err != nil {
			seedErr = err
		}
	}

	alice, bob, carol, dave, emma := ids["alice"], ids["bob"], ids["carol"], ids["dave"], ids["emma"]

	// 林小敏 ↔ 陈杰：最后一条未读，用于展示未读角标
	conv1 := createConv("direct", "", 0, []int64{alice, bob})
	addMsg(conv1, bob, "小敏，今晚有空一起吃饭吗？", ago(185))
	addMsg(conv1, alice, "好呀，去哪家？", ago(180))
	addMsg(conv1, bob, "公司楼下那家川菜，七点？", ago(175))
	readAt := addMsg(conv1, alice, "没问题，我订个位子 👌", ago(170))
	lastBob := addMsg(conv1, bob, "我到了，在门口等你", ago(25))
	setRead(conv1, alice, readAt)
	setRead(conv1, bob, lastBob)

	// 林小敏 ↔ 苏晴
	conv2 := createConv("direct", "", 0, []int64{alice, carol})
	addMsg(conv2, carol, "周末去爬山吗？天气预报说晴天", ago(1440))
	addMsg(conv2, alice, "去！正好新买的登山鞋到了", ago(1435))
	addMsg(conv2, carol, "那我把路线发群里", ago(1380))
	readAt2 := addMsg(conv2, alice, "好嘞～", ago(1378))
	setRead(conv2, alice, readAt2)
	setRead(conv2, carol, readAt2)

	// 群聊：周末爬山小队
	conv3 := createConv("group", "周末爬山小队", alice, []int64{alice, bob, carol, dave})
	addMsg(conv3, carol, "这周六早上 8 点老地方集合，别迟到～", ago(1320))
	groupRead := addMsg(conv3, dave, "收到，我带点水果", ago(1260))
	addMsg(conv3, bob, "我开车，可以捎两个人", ago(1200))
	addMsg(conv3, alice, "那我带咖啡 ☕", ago(1140))
	addMsg(conv3, carol, "记得穿防滑的鞋！", ago(1100))
	setRead(conv3, alice, groupRead)
	setRead(conv3, bob, groupRead)
	setRead(conv3, dave, groupRead)
	setRead(conv3, carol, groupRead)

	addMoment := func(userID int64, content string, images []string, createdAt int64) int64 {
		if seedErr != nil {
			return 0
		}
		encoded := "[]"
		if len(images) > 0 {
			// 种子图片是写死的站内路径，手工拼接 JSON 已足够，无需再走编码。
			encoded = "["
			for i, img := range images {
				if i > 0 {
					encoded += ","
				}
				encoded += `"` + img + `"`
			}
			encoded += "]"
		}
		res, err := tx.Exec(
			`INSERT INTO moments (user_id, content, images, created_at) VALUES (?, ?, ?, ?)`,
			userID, content, encoded, createdAt,
		)
		if err != nil {
			seedErr = err
			return 0
		}
		id, _ := res.LastInsertId()
		return id
	}

	addLike := func(momentID, userID int64, createdAt int64) {
		if seedErr != nil {
			return
		}
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO moment_likes (moment_id, user_id, created_at) VALUES (?, ?, ?)`,
			momentID, userID, createdAt,
		); err != nil {
			seedErr = err
		}
	}

	addComment := func(momentID, userID int64, content string, createdAt int64) {
		if seedErr != nil {
			return
		}
		if _, err := tx.Exec(
			`INSERT INTO moment_comments (moment_id, user_id, content, created_at) VALUES (?, ?, ?, ?)`,
			momentID, userID, content, createdAt,
		); err != nil {
			seedErr = err
		}
	}

	// 朋友圈
	m1 := addMoment(bob, "周末爬山，山顶的风也太舒服了！下次还来 ⛰️",
		[]string{"/uploads/seed/hike.jpg", "/uploads/seed/summit.jpg"}, ago(300))
	addLike(m1, alice, ago(290))
	addLike(m1, carol, ago(280))
	addComment(m1, carol, "下次带我一个！", ago(270))
	addComment(m1, bob, "@苏晴 说定了，下周六出发", ago(260))

	m2 := addMoment(carol, "新学的拿铁拉花，勉强能看 ☕",
		[]string{"/uploads/seed/latte.jpg"}, ago(1560))
	addLike(m2, alice, ago(1500))
	addLike(m2, bob, ago(1400))
	addComment(m2, alice, "已经很棒啦！", ago(1450))

	m3 := addMoment(alice, "今天也是认真生活的一天。✍️",
		nil, ago(1800))
	addLike(m3, carol, ago(1700))
	addLike(m3, dave, ago(1600))

	m4 := addMoment(dave, "阳台上的多肉又胖了一圈 🌵",
		[]string{"/uploads/seed/succulent.jpg"}, ago(3000))
	addLike(m4, alice, ago(2900))
	addLike(m4, emma, ago(2800))

	_ = emma

	if seedErr != nil {
		return seedErr
	}
	return tx.Commit()
}
