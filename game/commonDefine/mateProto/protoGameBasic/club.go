package protoGameBasic

import (
	"time"
)

// 创建 俱乐部桌子
const ID_ClubCreateTable = 301

type CS_ClubCreateTable struct {
	PlayerClubID int32   `json:"clubID"`     // 玩家俱乐部ID
	ClubPlayID   int64   `json:"clubPlayID"` // 俱乐部玩法ID
	IP           string  `json:"ip"`         // 玩家IP
	Longitude    float64 `json:"lng"`        // 玩家经度
	Latitude     float64 `json:"lat"`        // 玩家纬度

	// 服务之间 传递(与客户端无关)
	MZClubID        int32   `json:"mzClubID"`        // 盟主俱乐部ID
	PayUID          int64   `json:"payUID"`          // 支付人俄ID
	GameID          int32   `json:"playID"`          // 游戏本身ID
	PlayConfig      string  `json:"playOpt"`         // 玩法 配置
	TableConfig     string  `json:"tableCfg"`        // 桌子 配置
	ClubConfig      string  `json:"clubCfg"`         // 圈子玩法 配置
	PlayerClubScore float64 `json:"playerClubScore"` // 玩家俱乐部分
	IsStop3Players  bool    `json:"isStop3"`         // 是否禁止3人局
	IsStop4Players  bool    `json:"isStop4"`         // 是否禁止4人局
	MaxTZCount      int     `json:"maxTZC"`          // 最大同桌数

	RobotJoinReady   int32 `json:"robotR1"` //
	RobotJoinPlaying int32 `json:"robotR2"` //
	RobotInviteTimer int64 `json:"robotR3"` //
}

// 加入 俱乐部桌子
const ID_ClubJoinTable = 302

type CS_ClubJoinTable struct {
	TableNumber int32   `json:"tableNum"` // 桌子编号
	IP          string  `json:"ip"`       // 玩家IP
	Longitude   float64 `json:"lng"`      // 玩家经度
	Latitude    float64 `json:"lat"`      // 玩家纬度

	// 服务之间 传递(与客户端无关)
	ClubID          int32   `json:"clubID"`          // 玩家俱乐部ID
	PlayerClubScore float64 `json:"playerClubScore"` // 玩家俱乐部分
}
type SC_JoinTable struct {
	GameID     int32 `json:"gameID"`
	ClubPlayID int64 `json:"playID"`
	TableID    int32 `json:"tableID"`
}

// 新建 俱乐部玩法
const ID_HelpPutClubPlay = 310

type CS_PutClubPlay struct {
	ClubID int32 `json:"clubID"` // 俱乐部ID

	ClubPlayID   int64  `json:"clubPlayID"` // 俱乐部玩法ID
	ClubPlayName string `json:"name"`       // 俱乐部玩法名称
	PlayID       int32  `json:"playID"`     // 玩法(游戏)本身ID
	PlayCfg      string `json:"playOpt" `   // 玩法(游戏)配置
	TableCfg     string `json:"tableCfg"`   // 桌子规则
	ClubCfgText  string `json:"clubRule"`   // 俱乐部规则
}

// 发送给对应的游戏服 检查参数
const ID_PutClubPlay_RPC = 314

const ID_ForceDissolveTable = 335

type CS_ForceDissolveTable struct {
	OperationClubID int32 `json:"operClubID"` // 操作者俱乐部ID
	TableID         int32 `json:"tableID"`
}

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
	//CreateTime  time.Time `json:"createTime"`
	Status    int32  `json:"status"` // 1:在玩
	ServiceID string `json:"serviceID"`
}
type SS_PushAllTable struct {
	TableArr []PushTable `json:"tables"`
}

// 删除某服务的所有桌子
const ID_DeleteServiceIDTable = 406

type SS_DeleteServiceIDTable struct {
	ServiceID string `json:"serviceID"`
}

// 观看者离开
const ID_LookerLeave = 407
