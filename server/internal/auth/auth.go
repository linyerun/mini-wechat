// Package auth 负责密码哈希（bcrypt）与 JWT 令牌的签发和校验。
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// claims 在标准 JWT 声明之外携带用户 ID。
type claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

// HashPassword 生成 bcrypt 哈希，盐值由库自动随机并内嵌在结果中。
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword 校验明文密码是否匹配哈希。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// NewToken 使用 HS256 签发带过期时间的令牌。
func NewToken(secret string, userID int64, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "mini-wechat",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
}

// ParseToken 校验签名与有效期，返回令牌中的用户 ID。
func ParseToken(secret, token string) (int64, error) {
	if token == "" {
		return 0, errors.New("missing token")
	}
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		// 拒绝非 HMAC 算法，防止 alg=none 等降级攻击。
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return 0, errors.New("invalid token")
	}
	return c.UserID, nil
}
