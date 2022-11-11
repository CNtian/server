package mjXZDDTable

import (
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
)

// 其他玩家信息
type MsgSeatData struct {
	UID          int64                           `json:"uid"` // 玩家id
	Nick         string                          `json:"nick"`
	HeadURL      string                          `json:"headURL"`
	Sex          int32                           `json:"sex"`
	IP           string                          `json:"ip"`
	IsGPS        bool                            `json:"gps"`
	ChangePai    []int8                          `json:"RchangeP"`
	ReadyDingQue int8                            `json:"Rque"`
	SeatNumber   int32                           `json:"seatNum"`
	SeatStatus   uint32                          `json:"seatStatus"`
	ClubID       int32                           `json:"cID"`
	ClubScore    string                          `json:"clubScore"` // 俱乐部分
	SeatScore    string                          `json:"seatScore"`
	RoundScore   string                          `json:"roundScore"`
	ShouPaiCount int32                           `json:"shouPaiCount"`
	ChuPai       []int8                          `json:"chuPai"`
	OperationPai []*gameMaJiang.OperationPaiInfo `json:"operPai"`
	DingQue      int8                            `json:"dingQue"` // 定缺
	ChanedPai    bool                            `json:"chaned"`
	HuOrder      int32                           `json:"order"`
	HuPai        int8                            `json:"huPai"`
	HuMode       int32                           `json:"huMode"` // 胡的方式

	VoteStatus int32 `json:"vote"` // 解散桌子 投票
	//OperationTime int64 `json:"operTime"` // 操作剩余时间
}

// 桌子数据
const ID_TableData = 20100

type SC_TableData struct {
	TableNumber      int32 `json:"tableNum"`    // 房间编号
	TableStatus      int32 `json:"tableStatus"` // 桌子状态
	BankerSeatNumber int32 `json:"banker"`      // 庄家座位号
	MZCID            int32 `json:"mzID"`        // 盟主ID

	RoundCount    int32  `json:"curRound"`  // 当前玩局数
	TableRuleText string `json:"tableRule"` // 桌子配置JSON
	ClubRuleText  string `json:"clubRule"`  // 俱乐部配置JSON

	CurPlayCard        int32          `json:"curPlayPai"`        // 最近出的牌
	CurPlaySeatNumber  int32          `json:"curPlaySeatNum"`    // 最近出牌的座位号
	CurMoPaiSeatNumber int32          `json:"curMoPaiSeatNum"`   // 最近摸牌的座位号
	CurPengSeatNumber  int32          `json:"curPengPaiSeatNum"` // 最近碰牌的座位号
	RemainCardCount    int32          `json:"remainCardCount"`   // 牌敦剩余数量
	SeatData           []*MsgSeatData `json:"seatData"`          // 座位上的数据
	ShouPai            []int8         `json:"shouPai"`           // 手牌
	OperationID        string         `json:"operationID"`       // 操作ID
	CurMoPai           int8           `json:"curMoPai"`          // 该座位当前摸的牌
	OperationItem      uint32         `json:"operItem"`          // 操作项 (吃碰杠胡)
	GangArr            []int8         `json:"gang"`              // 目前可杠时,可杠的牌
	AnGangCard         []int8         `json:"anGangPai"`         // 已经暗杠牌
	ClubScore          string         `json:"clubScore"`         // 俱乐部分
	GameRuleText       string         `json:"gameRule"`          // 游戏规则JSON
	FirstRoundReadTime int64          `json:"FRRT"`              // 首局准备的时间

	ChangePai []int8 `json:"changePai"`
	DingQue   int8   `json:"dingQue"` // 定缺

	DissolveID          int32 `json:"dissolveID"`     // 解散发起人
	LaunchDissolveTime  int64 `json:"dissolveTime"`   // 发起解散时,时间戳
	PlayerOperationTime int64 `json:"playerOperTime"` // 玩家操作时间倒计时
}

// 游戏 小局结束
const ID_RoundGameOver = 20101

type HuPaiSeat struct {
	UID           int64                           `json:"uid"`
	NickName      string                          `json:"nick"`
	Head          string                          `json:"head"`
	SeatNumber    int32                           `json:"seatNum"` // 座位号
	HuMode        int32                           `json:"huMode"`  // 0:没有胡 1:自摸 2:点炮
	HuPai         int8                            `json:"huPai"`   // 胡的 牌型 组合
	ShouPai       []int8                          `json:"shouPai"` // 手牌
	HuOrder       int32                           `json:"order"`
	OperationPai  []*gameMaJiang.OperationPaiInfo `json:"operPai"`       // 操作区域的牌
	HuPaiXing     []*gameMaJiang.HuPaiXing        `json:"huFS"`          // 胡的番数
	GangScore     string                          `json:"gangScore"`     // 杠
	GameScoreStep []qpTable.GameScoreRec          `json:"gameScoreStep"` // 游戏中的 得分记录

	RoundScore string `json:"roundScore"` // 游戏输赢分
	SeatScore  string `json:"seatScore"`  // 座位分
}
type BroadRoundGameOver struct {
	TableNumber        int32        `json:"tableNum"`        // 房间编号
	CurRoundCount      int32        `json:"curRoundCount"`   // 当前局数
	MaxRoundCount      int32        `json:"maxRoundCount"`   // 最大局数
	BankerSeatNumber   int32        `json:"banker"`          // 庄家座位号
	Hupai              int8         `json:"hupai"`           // 胡的牌
	CurPlayCard        int8         `json:"curPlayPai"`      // 最近出的牌
	CurPlaySeatNumber  int32        `json:"curPlaySeatNum"`  // 最近出牌的座位号
	CurMoPaiSeatNumber int32        `json:"curMoPaiSeatNum"` // 最近摸牌的座位号
	RemainCardCount    int32        `json:"remainCardCount"` // 牌敦剩余数量
	SeatDataArr        []*HuPaiSeat `json:"huPaiSeat"`       // 座位上的数据
	Timestamp          int64        `json:"timestamp"`       // 结束时间
}

// 游戏 大局结束
const ID_GameOver = 20102

type GameOverSeatData struct {
	ClubID       int32  `json:"clubID"`
	UID          int64  `json:"uid"`
	NickName     string `json:"nick"`
	Head         string `json:"head"`
	GangCount    int32  `json:"gangC"`    // 杠牌次数
	DianPaoCount int32  `json:"dianPaoC"` // 点炮次数
	JiePaoCount  int32  `json:"jiePaoC"`  // 接炮次数
	HuPaiCount   int32  `json:"huPaiC"`   // 胡牌次数
	SeatScore    string `json:"score"`    // 座位分数
	SeatScoreInt int64  `json:"-"`
	IsMaxWin     bool   `json:"isWin"` // 大赢家
}
type BroadGameOverData struct {
	TableNumber   int32 `json:"tableNum"`      // 房间编号
	CurRoundCount int32 `json:"curRoundCount"` // 当前局数
	MaxRoundCount int32 `json:"maxRoundCount"` // 最大局数
	Timestamp     int64 `json:"timestamp"`     // 结束时间
	DissolveType  int32 `json:"dissolve"`      // 解散类型
	ClubID        int32 `json:"clubID"`        // 盟主圈子ID
	ClubPlayID    int64 `json:"clubPlayID"`    // 盟主玩法ID

	SeatData MJGameOverSeat `json:"seat"` // 座位
}
type MJGameOverSeat []*GameOverSeatData

func (s MJGameOverSeat) Len() int      { return len(s) }
func (s MJGameOverSeat) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s MJGameOverSeat) Less(i, j int) bool {
	return s[i].SeatScoreInt > s[j].SeatScoreInt
}

// 通知定缺
const ID_NoticeDingQue = 22103

type SC_NoticeDingQue struct {
	Pai int8
}

// 玩家定缺
const ID_PlayerDingQue = 22104

type CS_PlayerDingQue struct {
	Value int8 `json:"value"` // t:0 s:1 w:2 5
}

// 定缺已确认
const ID_BroadcastDingQueFinish = 22105

type CS_BroadcastDingQueFinish struct {
	Value int32 `json:"seatNum"`
}

// 广播玩家定缺结果
const ID_BroadcastPlayerDingQue = 22106

type DingQueValue struct {
	SeatNum int32 `json:"seatNum"`
	Value   int8  `json:"value"`
}
type SC_BroadcastPlayerDingQue struct {
	SeatArr []DingQueValue `json:"dingQue"`
}

// 通知换牌
const ID_NoticeChangePai = 22222

type SC_NoticeChangePai struct {
	Pai []int8
}

// 玩家换牌
const ID_PlayerChanePai = 22223

type CS_PlayerChanePai struct {
	Value []int8 `json:"value"`
}

// 换牌已确认
const ID_BroadcastChanePaiFinish = 22224

type CS_BroadcastChanePaiFinish struct {
	Value int32 `json:"seatNum"`
}

// 换牌结果
const ID_ChangePaiResult = 22225

type CS_ChangePaiResult struct {
	Value []int8 `json:"value"`
}

// 广播换牌座位号
const ID_BroadCastChangePaiResult = 22226

type CS_BroadCastChangePaiResult struct {
	Mode       int     `json:"mode"` // 0:对换  1:顺时针  2:逆时针
	SeatNumArr []int32 `json:"seat"`
}

// 广播胡
const ID_BroadcastHu = 22227

type SC_BroadcastHu struct {
	HuSeatNum int32 `json:"seatNum"`
	HuPai     int8  `json:"pai"`
	HuMode    int32 `json:"huMode"` // 0:没有胡 1:自摸 2:点炮 3:抢杠胡 4:点杠花

	GangSeat int32 `json:"gangSeat"`
	HuOrder  int32 `json:"huOrder"`
}

// 胡牌后 分数变化
const ID_HuNoticeChangeScore = 22228

type LoseSeat struct {
	Num   int32  `json:"n"` // 输分座位号
	Score string `json:"s"` // 变化分数
}
type BroadcastHuNoticeChangeScore struct {
	WinnerSeatNum int32  `json:"winNum"` // 得分座位号
	WinScore      string `json:"winS"`   // 得分

	LoserSeatArr []LoseSeat `json:"loser"`
}

// 呼叫转移
const ID_HuJiaoZhuanYi = 22229

type BroadcastHuJiaoZhuanYi struct {
	WinSeatArr  []LoseSeat `json:"win"`
	LoseSeatNum int32      `json:"loseNum"` // 输分座位号
	LoseScore   string     `json:"loseS"`   // 输分
}

// 自定义下一张牌
const ID_CustomNextPai = 20107

type CS_CustomNextPai struct {
	Pai int8 `json:"pai"` // 牌
}

// 获取剩余牌的
const ID_GetRemainingPai = 20108
