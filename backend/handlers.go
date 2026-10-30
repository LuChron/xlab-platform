package main

import (
	"net/http"
	"strings"

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
		// 申请正文、负责人和借用人属于业务隐私：访客不返回，团队只返回自己的记录。
		// 管理方负责评审与交接，保留全量视图。项目与资源台账仍作为公开展示数据。
		user, loggedIn := currentUser(c)
		if !loggedIn || user.Role != RoleAdmin {
			applications := make([]ProjectApplication, 0)
			borrows := make([]BorrowRecord, 0)
			if loggedIn && user.Role == RoleMember {
				for _, app := range l.Applications {
					if app.LeadID == user.TeamID {
						applications = append(applications, app)
					}
				}
				for _, borrow := range l.Borrows {
					if borrow.BorrowerID == user.TeamID {
						borrows = append(borrows, borrow)
					}
				}
			}
			l.Applications = applications
			l.Borrows = borrows

			// 活动流水同样携带业务正文（项目名、借用人等），必须与私有记录
			// 同一套可见性规则：团队只保留自己单据（含关联项目）的流水与
			// 台账级全局事件；访客保留公开可核验字段但隐去业务正文。
			if loggedIn && user.Role == RoleMember {
				ids := map[string]bool{}
				for _, app := range l.Applications {
					if app.LeadID == user.TeamID {
						ids[app.ID] = true
					}
				}
				for _, p := range l.Projects {
					if p.LeadID == user.TeamID {
						ids[p.ID] = true
					}
				}
				for _, borrow := range l.Borrows {
					if borrow.BorrowerID == user.TeamID {
						ids[borrow.ID] = true
					}
				}
				activities := make([]Activity, 0, len(l.Activities))
				for _, a := range l.Activities {
					if private := a.ObjectType == "application" || a.ObjectType == "borrow" || a.ObjectType == "project"; private && !ids[a.ObjectID] {
						continue
					}
					activities = append(activities, a)
				}
				l.Activities = activities
			} else {
				activities := make([]Activity, len(l.Activities))
				for i, a := range l.Activities {
					a.Detail = ""
					activities[i] = a
				}
				l.Activities = activities
			}
		}
		c.JSON(http.StatusOK, l)
	}
}

func historyHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !canReadHistory(c, store, c.Param("type"), c.Param("id")) {
			return
		}
		entries, err := store.History(c.Param("type"), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, entries)
	}
}

func canReadHistory(c *gin.Context, store Store, objectType, id string) bool {
	kind := strings.ToLower(objectType)
	// application/borrow/project 历史携带业务正文，必须校验归属；
	// equipment/space/allocation 属公开台账，任何登录用户可读。
	user, ok := currentUser(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
		return false
	}
	if user.Role == RoleAdmin {
		return true
	}
	if kind != "application" && kind != "borrow" && kind != "project" {
		return true
	}
	if user.Role != RoleMember {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权查看该记录"})
		return false
	}
	ledger, err := store.Ledger()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "无法校验记录归属：" + err.Error()})
		return false
	}
	if kind == "application" {
		for _, app := range ledger.Applications {
			if app.ID == id {
				if app.LeadID == user.TeamID {
					return true
				}
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权查看其他团队的申请历史"})
				return false
			}
		}
	} else if kind == "project" {
		for _, project := range ledger.Projects {
			if project.ID == id {
				if project.LeadID == user.TeamID {
					return true
				}
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权查看其他团队的项目历史"})
				return false
			}
		}
	} else {
		for _, borrow := range ledger.Borrows {
			if borrow.ID == id {
				if borrow.BorrowerID == user.TeamID {
					return true
				}
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权查看其他团队的借用历史"})
				return false
			}
		}
	}
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在"})
	return false
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
