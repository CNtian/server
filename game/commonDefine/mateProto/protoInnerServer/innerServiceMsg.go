package protoInnerServer

// 游戏服务器启动
const ID_GameSignIn = 110

type SignInInfo struct {
	PlayID   int32  `json:"playID"`
	PlayName string `json:"playName"`
}

type MsgGameSignIn struct {
	Status           int32        `json:"status"`
	SupportPlayIDArr []SignInInfo `json:"supportPlayID"`
	Port_pprof       string       `json:"pprof"`
}

// 大厅服务启动
const ID_HallServiceLaunch = 111

// 广播游戏服务状态
const ID_BroadGameServiceStatus = 112

type MsgBroadGameServiceStatus struct {
	Status     int32 `json:"status"`     // 0:运行中  1:不能创建新的桌子 2:不能加入桌子  3:停止游戏服务
	TableTotal int32 `json:"tableTotal"` // 桌子总数
	//PlayTableMap map[int32]int32 `json:"playTableTotal"` // key:玩法  value:桌子数量
}

// 真实玩家坐下
const ID_CallRobotComeIn = 113

type MsgCallRobotComeIn struct {
	GameID   int32 `json:"gameID"`
	TableID  int32 `json:"tableID"`
	MZClubID int32 `json:"mzClubID"`
	PlayID   int64 `json:"playID"`
}

// 俱乐部服务启动
const ID_ClubServiceLaunch = 120

// 通知在游戏中
const ID_NotiePlayerInTable = 133
