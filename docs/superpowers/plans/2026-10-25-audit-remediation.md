# X-Lab Audit Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use test-driven development and implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复审查确认的高、中优先级业务一致性、授权、并发和部署问题。

**Architecture:** HTTP 层用登录账号的 teamId 做对象归属校验，链码层用 Fabric clientID 做不可伪造的归属校验。demo 与 Fabric 共用同一组状态机约束；Fabric Gateway 连接缓存采用共享锁，身份选择失败直接返回错误。

**Tech Stack:** Go 1.21、Gin、Hyperledger Fabric Contract API、Vue 3、Bash

**Spec:** `docs/superpowers/specs/2026-10-25-audit-remediation-design.md`

## Global Constraints

- 不增加外部运行时依赖或新页面。
- demo 与 Fabric 对容量、异常结项和对象归属保持一致。
- 用户可填写的姓名只作显示，授权使用后端会话和 Fabric 证书身份。
- 普通 `up` 不删除账本，只有显式 `reset` 可清理网络。

## Review Focus

- 旧账本对象缺少 ownerIdentity 时必须拒绝写操作，而不是放行。
- 同一空间编号重复提交不能重复计算容量。
- Org2 连接并发初始化不能重复写共享 map 或回退到 Org1。
- EXCEPTION 解决前阻止结项，RESOLVED 后不再阻止。
- 相同时间戳的活动排序必须稳定。

---

### Task 1: 链码初始化与业务状态约束

**Files:**
- Create: `chaincode/contract_test.go`
- Modify: `chaincode/contract.go`
- Modify: `chaincode/vendor/`（测试所需 shimtest 包）

**Interfaces:**
- Produces: `requireMember`, `requireOwner`, 初始化标记、容量校验和按时间排序的活动查询。

- [x] 写链码失败测试：Org2 初始化、重复初始化、容量不足、EXCEPTION 结项、跨身份成果/借用/归还、活动时间排序。
- [x] 运行 `go test -mod=mod ./...`，确认测试因现有缺陷失败。
- [x] 实现最小链码修改并运行测试至通过。
- [x] 确认现有 vendor 已包含测试依赖，默认 `go test ./...` 通过。
- [x] 纳入最终修复提交。

### Task 2: demo 规则与 HTTP 对象归属

**Files:**
- Create: `backend/audit_regression_test.go`
- Modify: `backend/auth.go`
- Modify: `backend/main.go`
- Modify: `backend/handlers.go`
- Modify: `backend/store.go`
- Modify: `backend/types.go`
- Modify: `backend/seed.go`

**Interfaces:**
- Produces: `User.TeamID`、请求级团队授权、`BorrowerID`，以及 demo 容量与异常结项规则。

- [x] 写失败测试：容量不足、EXCEPTION 结项、跨团队成果、跨团队借用、非借用团队归还、表单身份被后端覆盖。
- [x] 运行后端测试，确认测试因现有缺陷失败。
- [x] 实现会话团队绑定、成员路由与对象归属检查。
- [x] 实现 demo 容量和 EXCEPTION 规则并运行全部后端测试。
- [x] 纳入最终修复提交。

### Task 3: Fabric Store 身份选择与并发安全

**Files:**
- Modify: `backend/audit_regression_test.go`
- Modify: `backend/store.go`
- Modify: `backend/auth.go`
- Modify: `backend/handlers.go`
- Modify: `backend/fabric.go`

**Interfaces:**
- Changes: `ActorAware.For(mspID string) (Store, error)`。
- Produces: 请求局部 Store；身份连接失败返回 503；共享 peers map 由同一把 RWMutex 保护。

- [x] 写失败测试：ActorAware 返回错误时处理器返回 503，且不调用原 Store 写方法。
- [x] 运行测试确认失败。
- [x] 修改 ActorAware、storeFor 和全部写处理器，禁止修改闭包捕获的 Store。
- [x] 为 Fabric peers 缓存添加共享锁并运行 `go test -race ./...`。
- [x] 纳入最终修复提交。

### Task 4: 输入模型、前端身份展示与部署语义

**Files:**
- Modify: `backend/store.go`
- Modify: `backend/fabric.go`
- Modify: `frontend/src/views/ResidencyView.vue`
- Modify: `frontend/src/views/EquipmentView.vue`
- Modify: `deploy/up.sh`
- Modify: `README.md`

**Interfaces:**
- Removes: `ApplicationInput.IdeaText`。
- Adds: `./deploy/up.sh reset`。

- [x] 删除独立 ideaText 输入，统一对项目说明计算哈希。
- [x] 前端负责人和借用人显示当前登录身份并设为只读。
- [x] 修改部署脚本：up 保留网络，reset 显式清理；运行 `bash -n deploy/up.sh`。
- [x] 更新 README 启动与数据说明。
- [x] 纳入最终修复提交。

### Task 5: 项目材料对齐

**Files:**
- Modify: `docs/答辩PPT.md`
- Modify: `docs/项目介绍.md`
- Modify: `docs/组员分工.md`
- Modify: `docs/演示视频素材/README.md`

**Interfaces:**
- Produces: 与实现一致的项目说明和明确的人工待办。

- [x] 将“双方确认”改为组织签名留痕和状态机校验，不声称链下正文已落地。
- [x] 把 18 处空间、28 台设备标为原型预置数据。
- [x] 将贡献评分改为每名成员 10 分制。
- [x] 保留姓名、分值、成品 PPT 和视频配音为人工提交项，不填造数据。
- [x] 纳入最终修复提交。

### Task 6: 全量验证

**Files:**
- Modify if needed: tests and documentation only for verified discrepancies.

**Interfaces:**
- Consumes: Tasks 1–5 outputs.
- Produces: 可提交的已验证仓库状态。

- [x] 运行 `go test -race ./...`（backend）。
- [x] 运行 `go test ./...`（chaincode）。
- [x] 运行 `./deploy/check.sh`。
- [x] 运行敏感信息扫描与 `git diff --check`。
- [x] 复查 README、项目介绍和答辩稿不存在实现不一致。
- [x] 提交最终验证修订并推送 main。
