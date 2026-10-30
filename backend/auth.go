package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 角色与组织映射：
//   admin  → Org1MSP  X-Lab 资源管理委员会（可评审、分配空间、审批与交接确认）
//   member → Org2MSP  入驻项目团队（提交申请、发起归还、提交成果）

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// User 演示账号
type User struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	TeamID      string `json:"teamId"`
	SubjectName string `json:"subjectName"`
	Role        string `json:"role"`
	Org         string `json:"org"`
	OrgLabel    string `json:"orgLabel"`
	Title       string `json:"title"`
}

// Account 带口令的账号记录（演示用明文，正式部署应存 bcrypt 哈希）
type Account struct {
	User
	Password string
}

func demoAccounts() []Account {
	return []Account{
		{
			User: User{
				Username: "admin", DisplayName: "资源管理委员会", Role: RoleAdmin,
				SubjectName: "资源管理委员会",
				Org:         "Org1MSP", OrgLabel: "X-Lab 资源管理委员会",
				Title: "空间与设备管理",
			},
			Password: "xlab2026",
		},
		{
			User: User{
				Username: "team", DisplayName: "交互式学习装置", Role: RoleMember,
				TeamID: "linxy", SubjectName: "林同学",
				Org: "Org2MSP", OrgLabel: "入驻项目团队",
				Title: "林同学 · 项目负责人",
			},
			Password: "xlab2026",
		},
	}
}

// Session 登录会话
type Session struct {
	Token     string
	User      User
	ExpiresAt time.Time
}

// AuthService 会话管理（进程内存；重启后需重新登录）
type AuthService struct {
	mu       sync.RWMutex
	accounts map[string]Account
	sessions map[string]Session
	ttl      time.Duration
}

func NewAuthService() *AuthService {
	a := &AuthService{
		accounts: map[string]Account{},
		sessions: map[string]Session{},
		ttl:      12 * time.Hour,
	}
	for _, acc := range demoAccounts() {
		a.accounts[acc.Username] = acc
	}
	return a
}

func (a *AuthService) Login(username, password string) (Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	acc, ok := a.accounts[strings.ToLower(strings.TrimSpace(username))]
	if !ok || acc.Password != password {
		return Session{}, errors.New("账号或密码不正确")
	}

	token, err := randomToken()
	if err != nil {
		return Session{}, errors.New("生成会话失败")
	}
	sess := Session{Token: token, User: acc.User, ExpiresAt: time.Now().Add(a.ttl)}
	a.sessions[token] = sess
	return sess, nil
}

func (a *AuthService) Logout(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, token)
}

// Resolve 校验 token 并返回会话；每次校验顺带清扫所有过期会话，
// 避免 12 小时 TTL 之后 sessions map 里的死会话无限累积。
func (a *AuthService) Resolve(token string) (Session, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for t, s := range a.sessions {
		if now.After(s.ExpiresAt) {
			delete(a.sessions, t)
		}
	}
	sess, ok := a.sessions[token]
	if !ok {
		return Session{}, false
	}
	return sess, true
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// ---------------- HTTP ----------------

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loginHandler(auth *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		sess, err := auth.Login(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": sess.Token, "user": sess.User})
	}
}

func logoutHandler(auth *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth.Logout(bearerToken(c))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func meHandler(auth *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sess, ok := auth.Resolve(bearerToken(c))
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "登录已过期"})
			return
		}
		c.JSON(http.StatusOK, sess.User)
	}
}

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if t := c.GetHeader("X-XLab-Token"); t != "" {
		return strings.TrimSpace(t)
	}
	return ""
}

// authRequired 校验登录，并把会话放进 context
func authRequired(auth *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sess, ok := auth.Resolve(bearerToken(c))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
			return
		}
		c.Set("session", sess)
		c.Next()
	}
}

// optionalAuth 用于既允许访客访问、又需要按身份裁剪数据的只读接口。
// 没有 token 时按访客处理；带了失效 token 则明确返回 401，避免前端继续保留过期身份。
func optionalAuth(auth *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token == "" {
			c.Next()
			return
		}
		sess, ok := auth.Resolve(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "登录已过期"})
			return
		}
		c.Set("session", sess)
		c.Next()
	}
}

// currentUser 从 context 取当前用户
func currentUser(c *gin.Context) (User, bool) {
	v, ok := c.Get("session")
	if !ok {
		return User{}, false
	}
	sess, ok := v.(Session)
	return sess.User, ok
}

// requireAdminRole 接口层的角色校验（链码层还会按组织身份再校验一次）
func requireAdminRole(c *gin.Context) bool {
	u, ok := currentUser(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
		return false
	}
	if u.Role != RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "权限不足：该操作仅允许资源管理委员会（Org1）执行，当前身份为" + u.OrgLabel,
		})
		return false
	}
	return true
}

// requireAdminRoleMiddleware gin 中间件版本
func requireAdminRoleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireAdminRole(c) {
			return
		}
		c.Next()
	}
}

func requireMemberRole(c *gin.Context) bool {
	u, ok := currentUser(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
		return false
	}
	if u.Role != RoleMember || strings.TrimSpace(u.TeamID) == "" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code": 403, "message": "权限不足：该操作仅允许已登记的入驻项目团队执行",
		})
		return false
	}
	return true
}

func requireMemberRoleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMemberRole(c) {
			return
		}
		c.Next()
	}
}

// storeFor 返回以当前登录用户组织身份签名的 Store。
// 写操作必须走这个入口：链码的 actorMsp 读的是交易证书，
// 团队发起的申请若由委员会私钥签名，账本上就查不出是谁提的。
func storeFor(store Store, c *gin.Context) (Store, error) {
	aware, ok := store.(ActorAware)
	if !ok {
		return store, nil
	}
	u, ok := currentUser(c)
	if !ok || u.Org == "" {
		return nil, errors.New("无法确定当前请求的组织身份")
	}
	return aware.For(u.Org)
}
