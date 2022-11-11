package protoGameBasic

// 应答所有非法操作
type JsonResponse struct {
	Status int32       `json:"status"`
	Detail string      `json:"detail"`
	Data   interface{} `json:"data"`
}

// 玩家请求坐下
const ID_ReqSitDown = 1050

type CS_ReqSitDown struct {
	SeatNumber int32 `json:"seatNum"` // <0:随机空位  >=0:指定座位

	// 服务器使用
	SeatScore int64 `json:"seatScore,omitempty"`
}
type SC_ReqSitDown struct {
	Status int32 `json:"status"`
}

// 玩家准备
const ID_GameReady = 1051

// 请求离开桌子
const ID_ReqLeaveTable = 1052

// 广播 玩家离开桌子
const ID_BroadPlayerLeaveTable = 1053

type BroadPlayerLeaveTable struct {
	UID     int64 `json:"uid"`
	SeatNum int32 `json:"seatNum"`
}

// 手动开始游戏
const ID_GameStart = 1054

// 广播玩家状态
const ID_BroadPlayerStatus = 1055

type BroadcastPlayerStatus struct {
	UID     int64  `json:"uid"`
	SeatNum int32  `json:"seatNum"`
	Status  uint32 `json:"status"`
}

// 广播新玩家 进入
const ID_NewPlayerJoin = 1056

type BroadcastNewPlayerJoin struct {
	UID       int64  `json:"uid"`
	SeatNum   int32  `json:"seatNum"`
	Status    uint32 `json:"status"`
	Head      string `json:"head"`
	Nick      string `json:"nick"`
	Sex       int32  `json:"sex"`
	IP        string `json:"ip"`
	Location  bool   `json:"gps"`
	ClubScore string `json:"clubScore"` // 俱乐部分
}

// 普通玩家 发起解散 桌子
const ID_LaunchDissolveTable = 1002

type BroadcastDissolveTableResult struct {
	SeatNum int32 `json:"seatNum"` // 座位号
}

// 解散桌子 玩家投票操作
const ID_DissolveTableVote = 1003

type CS_DissolveTableVote struct {
	Vote int32 `json:"vote"` // 1:同意  2:不同意
}

type BroadcastDissolveTableVoteResult struct {
	SeatNumber int32 `json:"seatNum"`
	Vote       int32 `json:"vote"` // 1:同意  2:不同意
}

// 解散 桌子(非游戏正常结束)
const ID_TableExpire = 1004

type CS_DissolveTable struct {
	TableNumber int32 `json:"tableNum"`   // 桌子编号
	Superpower  bool  `json:"superpower"` // 是否强制解散
}

// 玩家当局游戏分变化
const ID_PlayerRoundScoreChanged = 1005

type BroadcastPlayerScoreChanged struct {
	Category      int     `json:"category"` // 原因
	WinnerSeatNum int32   `json:"winNum"`   // 得分座位号
	LoserSeatNum  []int32 `json:"loseNum"`  // 输分座位号
	Score         string  `json:"score"`    // 变化分数
}

// 取消托管
const ID_CancelTrusteeship = 1006

// 投票处理结果
const ID_DissolveTableVoteReslut = 1007

type DissolveTableVoteResult struct {
	IsDissolveTable bool `json:"isDissolve"`
}

// 玩家互动
const ID_PlayerInteractive = 1008

type CS_PlayerInteractive struct {
	Type    int32  `json:"type"`
	Content string `json:"content"`
	To      int32  `json:"to"`
}

type SC_PlayerInteractive struct {
	SendSeatNum int32  `json:"sender"`
	To          int32  `json:"to"`
	Type        int32  `json:"type"`
	Content     string `json:"content"`
}

// 获取 小结算数据
const ID_GetRoundOverMsg = 1009

// GPS信息
const ID_GetGPSInfo = 1010

// 桌子状态变化
const ID_GameTableStatusChanged = 1011

type SC_GameTableStatusChanged struct {
	Status uint32 `json:"status"`
}

// 主动打开托管
const ID_ActiveTrusteeship = 1012

// 定时准备
const (
	TIMER_FirstRoundReady = 615 // 首局自动准备
	TIMER_DissolveTable   = 616
	TIMER_Leave           = 617
	TIMER_AutoRedy        = 618 // 开始后自动准备

	PaoDeKuai = 700 // 跑得快
	KaWuXing  = 705 // 卡五星(襄阳)
	NiuNiu    = 710
	ZhaJinHua = 715
)
