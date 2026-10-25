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
		if !requireMemberRole(c) {
			return
		}
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var in ApplicationInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		user, _ := currentUser(c)
		in.LeadID, in.LeadName = user.TeamID, user.SubjectName
		app, err := requestStore.CreateApplication(in)
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
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var req reviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if err := requestStore.ReviewApplication(c.Param("id"), req.Approved, req.Note); err != nil {
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
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var req activateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		proj, err := requestStore.ActivateProject(c.Param("id"), req.SpaceIDs)
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
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var req extendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if err := requestStore.ExtendProject(c.Param("id"), req.NewEndDate, req.Note); err != nil {
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
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var req noteRequest
		_ = c.ShouldBindJSON(&req)
		if err := requestStore.CloseProject(c.Param("id"), req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func milestoneHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMemberRole(c) {
			return
		}
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		user, _ := currentUser(c)
		if !requireProjectOwner(c, requestStore, c.Param("id"), user.TeamID) {
			return
		}
		var req noteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if err := requestStore.SubmitMilestone(c.Param("id"), req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ---------- 设备交接 ----------

func createBorrowHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMemberRole(c) {
			return
		}
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var in BorrowInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		user, _ := currentUser(c)
		if !requireProjectOwner(c, requestStore, in.ProjectID, user.TeamID) {
			return
		}
		in.BorrowerID, in.Borrower = user.TeamID, user.SubjectName
		rec, err := requestStore.RequestBorrow(in)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusCreated, rec)
	}
}

func reviewBorrowHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var req reviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if err := requestStore.ReviewBorrow(c.Param("id"), req.Approved, req.Note); err != nil {
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
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var req pickupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if req.Condition == "" {
			req.Condition = "GOOD"
		}
		if err := requestStore.ConfirmPickup(c.Param("id"), req.Condition, req.Photos); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func returnRequestHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMemberRole(c) {
			return
		}
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		user, _ := currentUser(c)
		if !requireBorrowOwner(c, requestStore, c.Param("id"), user.TeamID) {
			return
		}
		var req noteRequest
		_ = c.ShouldBindJSON(&req)
		if err := requestStore.RequestReturn(c.Param("id"), req.Note); err != nil {
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
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var req returnConfirmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求格式不正确"})
			return
		}
		if req.Condition == "" {
			req.Condition = "NORMAL"
		}
		if err := requestStore.ConfirmReturn(c.Param("id"), req.Condition, req.Note, req.Photos); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func resolveHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestStore, ok := selectRequestStore(c, store)
		if !ok {
			return
		}
		var req noteRequest
		_ = c.ShouldBindJSON(&req)
		if err := requestStore.ResolveException(c.Param("id"), req.Note); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func selectRequestStore(c *gin.Context, base Store) (Store, bool) {
	store, err := storeFor(base, c)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code": 503, "message": "当前组织的 Fabric 签名身份不可用：" + err.Error(),
		})
		return nil, false
	}
	return store, true
}

func requireProjectOwner(c *gin.Context, store Store, projectID, teamID string) bool {
	if projectID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 400, "message": "必须选择所属项目"})
		return false
	}
	ledger, err := store.Ledger()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "无法校验项目归属：" + err.Error()})
		return false
	}
	for _, project := range ledger.Projects {
		if project.ID == projectID {
			if project.LeadID != teamID {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权操作其他团队的项目"})
				return false
			}
			return true
		}
	}
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "message": "项目不存在"})
	return false
}

func requireBorrowOwner(c *gin.Context, store Store, borrowID, teamID string) bool {
	ledger, err := store.Ledger()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "无法校验借用单归属：" + err.Error()})
		return false
	}
	for _, borrow := range ledger.Borrows {
		if borrow.ID == borrowID {
			if borrow.BorrowerID != teamID {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权操作其他团队的借用单"})
				return false
			}
			return true
		}
	}
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "message": "借用单不存在"})
	return false
}
