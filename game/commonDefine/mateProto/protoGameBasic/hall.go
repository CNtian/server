package protoGameBasic

// 创建 私人桌子
const ID_PrivateCreateGameTable = 201

type CS_PrivateCreateGameTable struct {
	GameID      int32  `json:"playID"`
	PlayConfig  string `json:"playOpt"`
	TableConfig string `json:"tableCfg"`

	IP        string  `json:"ip"`  // 玩家IP
	Longitude float64 `json:"lng"` // 玩家经度
	Latitude  float64 `json:"lat"` // 玩家纬度
}

// 加入 私人桌子
const ID_PrivateJoinGameTable = 202

type CS_PrivateJoinGameTable struct {
	TableNumber int32 `json:"tableNumber"`

	IP        string  `json:"ip"`  // 玩家IP
	Longitude float64 `json:"lng"` // 玩家经度
	Latitude  float64 `json:"lat"` // 玩家纬度
}

// 改变游戏服务的状态
const ID_ChangeGameServiceStatus = 203

type SS_CMD struct {
	To     string `json:"to"`
	Status int32  `json:"status"`
}

// 添加新桌子
const ID_AddNewTable = 213

type SS_AddNewTable struct {
	GameID  int32
	TableID int32
	Source  string
}

// 删除新桌子
const ID_DeleteTable = 214

type SS_DeleteTable struct {
	TableID int32
	Players []int64
}

// 玩家进入
const ID_PlayerJoinTable = 215

type SS_PlayerJoinTable struct {
	TableID int32
}

// 玩家离开
const ID_PlayerLeaveTable = 216

type SS_PlayerLeaveTable struct {
	TableID int32
}

// 恢复桌子
const ID_HallRecoverTable = 217

type HallRecoverTable struct {
	GameID  int32
	TableID int32
	Players []int64
}
type SS_RecoverTable struct {
	Data []HallRecoverTable
}
