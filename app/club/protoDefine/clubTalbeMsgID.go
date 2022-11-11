package clubProto

import "time"

// 添加新桌子
const ID_TablePutNew = 400

type SS_PutNewTable struct {
	ClubID      int32     `json:"clubID"`
	TableNumber int32     `json:"tableNum"`
	ClubPlayID  int64     `json:"clubPlayID"`
	GameID      int32     `json:"gameID"`
	MaxPlayers  int32     `json:"maxPlayers"` // 最多人数
	UID         int64     `json:"uid"`
	CreateTime  time.Time `json:"createTime"`
	ServiceID   string    `json:"serviceID"`
}

// 桌子中添加 玩家
const ID_TablePutPlayer = 401

type SS_PutPlayerToTable struct {
	ClubID      int32 `json:"clubID"`
	GameID      int32 `json:"gameID"`
	TableNumber int32 `json:"tableNum"`
	UID         int64 `json:"uid"`
}

// 桌子中删除 玩家
const ID_TableDelPlayer = 402

type SS_DelPlayerInTable struct {
	ClubID      int32 `json:"clubID"`
	GameID      int32 `json:"gameID"`
	TableNumber int32 `json:"tableNum"`
	UID         int64 `json:"uid"`
}

// 桌子从 准备 变为 开始
const ID_TableStatusChanged = 403

type SS_TableStatusChanged struct {
	ClubID      int32 `json:"clubID"`
	GameID      int32 `json:"gameID"`
	TableNumber int32 `json:"tableNum"`
	Status      int32 `json:"status"` // 暂未使用
}

// 删除桌子
const ID_TableDelete = 404

type SS_DelTable struct {
	ClubID      int32 `json:"clubID"`
	GameID      int32 `json:"gameID"`
	TableNumber int32 `json:"tableNum"`
}

// 推送所有桌子(恢复)
const ID_PushAllTable = 405

type PushTable struct {
	ClubID      int32   `json:"clubID"`
	TableNumber int32   `json:"tableNum"`
	ClubPlayID  int64   `json:"clubPlayID"`
	GameID      int32   `json:"gameID"`
	MaxPlayers  int32   `json:"maxPlayers"` // 最多人数
	UIDArr      []int64 `json:"uid"`
	Status      int32   `json:"status"` // 1:在玩
	ServiceID   string  `json:"serviceID"`
}
type SS_PushAllTable struct {
	TableArr []PushTable `json:"tables"`
}

// 删除某服务的所有桌子
const ID_DeleteServiceIDTable = 406

type SS_DeleteServiceIDTable struct {
	ServiceID string `json:"serviceID"`
}

// 应答快速开始
const ID_ReplyQuickStart = 408

type SS_ReplyQuickStart struct {
	QuickStartData *CS_ClubQuickStart
	TableNum       int32
}

// 获取桌子数量
const ID_GetClubTableCount = 409

type SS_GetClubTableCount struct {
	ClubID int32
}

type ClubPlayTableCount struct {
	PlayID int64
	Count  int
}
type SSRSP_GetClubTableCount struct {
	ClubID int32

	Arr []ClubPlayTableCount
}
