package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	mode := envOr("XLAB_MODE", "auto") // auto | demo | fabric

	var store Store
	switch mode {
	case "demo":
		store = newMemoryStore()
		log.Println("[XLAB] 演示模式：使用内存账本，未连接 Fabric")
	case "fabric":
		fs, err := newFabricStore(fabricConfigFromEnv())
		if err != nil {
			log.Fatalf("[XLAB] 连接 Fabric 失败: %v", err)
		}
		if _, err := fs.Ledger(); err != nil {
			fs.Close()
			log.Fatalf("[XLAB] Fabric Gateway 已创建，但账本不可读: %v", err)
		}
		store = fs
		log.Println("[XLAB] 已连接 Fabric Gateway")
	default:
		fs, err := newFabricStore(fabricConfigFromEnv())
		if err == nil {
			_, err = fs.Ledger()
		}
		if err != nil {
			if fs != nil {
				fs.Close()
			}
			log.Printf("[XLAB] Fabric 不可用（%v），自动回退到演示模式", err)
			store = newMemoryStore()
		} else {
			store = fs
			log.Println("[XLAB] 已连接 Fabric Gateway")
		}
	}
	defer store.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	auth := NewAuthService()

	api := r.Group("/api")
	{
		// 公开接口
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true, "mode": store.Mode()})
		})
		api.POST("/auth/login", loginHandler(auth))

		// 公开只读：首页与链上记录对访客开放（链上数据本就可公开核验）
		api.GET("/ledger", ledgerHandler(store))
		api.GET("/history/:type/:id", historyHandler(store))
		api.GET("/chain", chainHandler(store))

		// 需要登录
		authed := api.Group("")
		authed.Use(authRequired(auth))
		{
			authed.GET("/auth/me", meHandler(auth))
			authed.POST("/auth/logout", logoutHandler(auth))

			// 项目申请、成果和设备借还由入驻团队发起。
			member := authed.Group("")
			member.Use(requireMemberRoleMiddleware())
			{
				member.POST("/applications", createApplicationHandler(store))
				member.POST("/projects/:id/milestone", milestoneHandler(store))
				member.POST("/borrows", createBorrowHandler(store))
				member.POST("/borrows/:id/return-request", returnRequestHandler(store))
			}

			// 项目入驻 —— 管理类：仅 Org1
			admin := authed.Group("")
			admin.Use(requireAdminRoleMiddleware())
			{
				admin.POST("/applications/:id/review", reviewApplicationHandler(store))
				admin.POST("/applications/:id/activate", activateHandler(store))
				admin.POST("/projects/:id/extend", extendHandler(store))
				admin.POST("/projects/:id/close", closeHandler(store))
				admin.POST("/borrows/:id/review", reviewBorrowHandler(store))
				admin.POST("/borrows/:id/pickup", pickupHandler(store))
				admin.POST("/borrows/:id/return-confirm", returnConfirmHandler(store))
				admin.POST("/borrows/:id/resolve", resolveHandler(store))
			}

		}
	}

	// 前端构建产物
	webDir := envOr("XLAB_WEB_DIR", filepath.Join("..", "frontend", "dist"))
	mountWeb(r, webDir)

	port := envOr("XLAB_PORT", "8088")
	log.Printf("[XLAB] 服务已启动 http://127.0.0.1:%s  (模式: %s)", port, store.Mode())
	if err := r.Run("127.0.0.1:" + port); err != nil {
		log.Fatal(err)
	}
}

// mountWeb 托管前端静态资源，并对非 API 路径回退到 index.html（SPA）
func mountWeb(r *gin.Engine, webDir string) {
	index := filepath.Join(webDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		log.Printf("[XLAB] 未找到前端构建产物 %s，仅提供 API", index)
		return
	}
	r.Static("/assets", filepath.Join(webDir, "assets"))
	// public/ 下的资源（照片、视频、logo）构建后落在 dist 根目录
	r.Static("/media", filepath.Join(webDir, "media"))
	r.StaticFile("/favicon.svg", filepath.Join(webDir, "favicon.svg"))
	r.StaticFile("/xlab-logo.svg", filepath.Join(webDir, "xlab-logo.svg"))
	r.StaticFile("/xlab-name.svg", filepath.Join(webDir, "xlab-name.svg"))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"message": "接口不存在"})
			return
		}
		c.File(index)
	})
}

// corsMiddleware 便于前端 dev server 直接联调
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-XLab-Role")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// fail 统一错误返回
func fail(c *gin.Context, err error) {
	c.JSON(http.StatusConflict, gin.H{"code": 409, "message": err.Error()})
}
