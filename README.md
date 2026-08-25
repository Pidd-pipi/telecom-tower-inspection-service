# Telecom Tower Inspection Service

通信塔结构巡检服务，覆盖塔站巡检任务排程、缺陷记录、维修工单流转、风险评分与运维工单状态机，并内置前端巡检页面。

## Layout

`backend/` 为 Go 模块（`go.mod` 在 `backend/` 内）：

- `httpapi/`：既有塔站集合与状态变更接口、健康检查、静态页面
- `store/`、`domain/`、`validation/`：塔站基础存储、模型与状态校验
- `tower_*.go`：巡检任务、缺陷、工单、风险评估、审计、导出、统计、报表业务层
- `ops_*.go`：运维工单状态机与记录管理（/ops 接口）
- `web/`：go:embed 前端页面（`index.html`、`app.js`）

## Run and API

从 `backend/` 执行 `go run .`，默认监听 `8080`，可设置 `PORT` 使用其他端口。

### 塔站基础接口

- `GET /healthz`：健康检查
- `GET /api/towers`：塔站集合
- `POST /api/towers/status`：更新塔站状态，body：`{"id":"TWR-204","status":"repair_required"}`
- `GET /`：巡检页面

### 巡检与缺陷接口

- `GET /api/inspections`：巡检任务列表，支持 `tower_id`、`region`、`status`、`page`、`page_size`
- `POST /api/inspections`：排程巡检，body：`{"tower_id":"TWR-118","inspector":"mei","region":"Kanto North","due_in_days":30}`
- `GET /api/inspections/{id}`：巡检详情
- `POST /api/inspections/{id}/start`、`POST /api/inspections/{id}/complete`：状态流转
- `GET /api/inspections/{id}/audit`：巡检事件审计
- `GET /api/findings`：缺陷列表，支持 `tower_id`、`severity`
- `POST /api/findings`：记录缺陷，body：`{"tower_id":"TWR-204","severity":"high","category":"loose_bolt","detail":"..."}`（high 缺陷自动生成维修工单）
- `GET /api/workorders`、`POST /api/workorders/{id}/assign`、`POST /api/workorders/{id}/resolve`：工单流转
- `GET /api/risk/{towerID}`：塔站风险评分
- `GET /api/summary`：统计汇总
- `POST /api/export`：批量导出巡检状态，body：`{"from":"in_progress","to":"completed"}`
- `GET /api/report`：巡检报表

### 运维工单状态机

- `GET /ops/records`：记录列表
- `POST /ops/records`：创建记录
- `GET /ops/records/{id}`：记录详情
- `POST /ops/records/{id}/transition`：状态流转
- `GET /ops/records/{id}/audit`：审计
- `GET /ops/snapshot`：快照统计
- `GET /ops/rules`：规则列表

## Verification

- `gofmt -w .`: passed
- `go build ./...`: passed
- `go test ./...`: passed
- Runtime smoke: health check 200，塔站接口与巡检接口正常返回。

## Enterprise Notes

请求保留请求标识并经过恢复与超时保护；巡检任务与运维记录使用版本校验与状态机流转；审计事件、统计缓存与导出 worker 保障并发安全；页面通过 fetch 调用集合与状态变更接口。
