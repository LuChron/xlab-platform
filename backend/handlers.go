package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 本文件集中放置 HTTP 处理器，业务规则全部委托给 Store，
// 保证“演示模式 / Fabric 模式”行为一致。

func ledgerHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		l, err := store.Ledger()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, l)
	}
}

func historyHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		entries, err := store.History(c.Param("type"), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, entries)
	}
}

func chainHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		info, err := store.Chain()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, info)
	}
}

// ---------- 项目入驻 ----------

func createApplicationHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var in ApplicationInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		app, err := store.CreateApplication(in)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusCreated, app)
	}
}

type reviewRequest struct {
	Approved bool   `json:"approved"`
	Note     string `json:"note"`
}

func reviewApplicationHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req reviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if err := store.ReviewApplication(c.Param("id"), req.Approved, req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

type activateRequest struct {
	SpaceIDs []string `json:"spaceIds"`
}

func activateHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req activateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		proj, err := store.ActivateProject(c.Param("id"), req.SpaceIDs)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusCreated, proj)
	}
}

type extendRequest struct {
	NewEndDate string `json:"newEndDate"`
	Note       string `json:"note"`
}

func extendHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req extendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if err := store.ExtendProject(c.Param("id"), req.NewEndDate, req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

type noteRequest struct {
	Note string `json:"note"`
}

func closeHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req noteRequest
		_ = c.ShouldBindJSON(&req)
		if err := store.CloseProject(c.Param("id"), req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func milestoneHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req noteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if err := store.SubmitMilestone(c.Param("id"), req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ---------- 设备交接 ----------

func createBorrowHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var in BorrowInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		rec, err := store.RequestBorrow(in)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusCreated, rec)
	}
}

func reviewBorrowHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req reviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if err := store.ReviewBorrow(c.Param("id"), req.Approved, req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

type pickupRequest struct {
	Condition string   `json:"condition"`
	Photos    []string `json:"photos"`
}

func pickupHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req pickupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if req.Condition == "" {
			req.Condition = "GOOD"
		}
		if err := store.ConfirmPickup(c.Param("id"), req.Condition, req.Photos); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func returnRequestHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req noteRequest
		_ = c.ShouldBindJSON(&req)
		if err := store.RequestReturn(c.Param("id"), req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

type returnConfirmRequest struct {
	Condition string   `json:"condition"`
	Note      string   `json:"note"`
	Photos    []string `json:"photos"`
}

func returnConfirmHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req returnConfirmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if req.Condition == "" {
			req.Condition = "NORMAL"
		}
		if err := store.ConfirmReturn(c.Param("id"), req.Condition, req.Note, req.Photos); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func resolveHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 按登录用户所属组织签名，链上才能记下真实的调用方
		store = storeFor(store, c)
		var req noteRequest
		_ = c.ShouldBindJSON(&req)
		if err := store.ResolveException(c.Param("id"), req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
