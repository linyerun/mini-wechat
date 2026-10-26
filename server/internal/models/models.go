// Package models 定义跨层共享的数据结构，JSON 标签即对外接口的字段命名。
package models

// User 用户资料；不包含密码哈希，可安全返回给前端。
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
}

// Message 会话消息。Type 为 text 或 image，image 时 Content 存图片相对路径。
type Message struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversationId"`
	SenderID       int64  `json:"senderId"`
	Type           string `json:"type"`
	Content        string `json:"content"`
	CreatedAt      int64  `json:"createdAt"`
	Sender         *User  `json:"sender,omitempty"`
}

// Conversation 会话视图。Type 为 direct 或 group，Unread 按查看者计算。
type Conversation struct {
	ID                int64    `json:"id"`
	Type              string   `json:"type"`
	Name              string   `json:"name"`
	OwnerID           int64    `json:"ownerId"`
	CreatedAt         int64    `json:"createdAt"`
	Members           []User   `json:"members"`
	LastMessage       *Message `json:"lastMessage"`
	Unread            int      `json:"unread"`
	LastReadMessageID int64    `json:"lastReadMessageId"`
}

// MomentComment 朋友圈评论。
type MomentComment struct {
	ID        int64  `json:"id"`
	User      User   `json:"user"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"createdAt"`
}

// Moment 朋友圈动态，Images/Likes/Comments 在查询时按查看者补全。
type Moment struct {
	ID        int64           `json:"id"`
	User      User            `json:"user"`
	Content   string          `json:"content"`
	Images    []string        `json:"images"`
	Likes     []User          `json:"likes"`
	Comments  []MomentComment `json:"comments"`
	CreatedAt int64           `json:"createdAt"`
	LikedByMe bool            `json:"likedByMe"`
}
