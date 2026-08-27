基于 Go 实现的城市地铁站台屏蔽门系统项目，一款轨道交通站台设备控制服务，完成屏蔽门开关门流程、车门联锁、防夹检测与站台门状态监控管理。

# PlatformScreenDoor

PlatformScreenDoor 是城市地铁站台屏蔽门系统的轨道交通站台设备控制服务。系统接收列车停稳与对准信号，按车门-屏蔽门编号映射完成联锁校验后下发开门；发车准备时联动关门，关门过程持续做防夹采样，夹到障碍物立即停止并告警；门体状态、告警与事件流水全部写入运行记录，值班员可通过控制台页面查看实时门状态。

## 构建

```bash
go build -mod=vendor ./...
```

## 运行

```bash
go run -mod=vendor ./cmd/server -addr 127.0.0.1:8090 -dir ./data
```

启动后访问 http://127.0.0.1:8090/ 打开控制台页面。

## HTTP 接口

- `GET /healthz` 健康检查
- `GET /api/doors` 门状态汇总
- `GET /api/events?after=N&limit=M` 事件流水（断点续传）
- `GET /api/alarms` 告警列表
- `GET /api/snapshot` 控制台快照
- `POST /api/door/open` 开门（受联锁约束），入参 `{doorID, trainID}`
- `POST /api/door/confirm` 开门到位确认
- `POST /api/door/close` 关门（防夹采样）
- `POST /api/door/reset` 防夹/急停复位
- `POST /api/console/local` 就地/自动切换
- `POST /api/heartbeat` 门控心跳上报
- `POST /api/train/dock` 列车停稳+对准信号
- `POST /api/train/leave` 列车离站

## 状态机

- 屏蔽门：closed -> opening -> open -> closing -> closed（防夹回 stopped）
- 联锁：idle -> interlocked -> released -> idle
- 列车状态：away -> docking -> docked -> departing -> away
