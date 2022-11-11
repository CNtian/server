package mj_XueZhan_KWXTable

import (
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
)

// 其他玩家信息
type MsgSeatData struct {
	UID            int64                           `json:"uid"` // 玩家id
	Nick           string                          `json:"nick"`
	HeadURL        string                          `json:"headURL"`
	Sex            int32                           `json:"sex"`
	IP             string                          `json:"ip"`
	IsGPS          bool                            `json:"gps"`
	SeatNumber     int32                           `json:"seatNum"`
	SeatStatus     uint32                          `json:"seatStatus"`
	ClubID         int32                           `json:"cID"`
	ClubScore      string                          `json:"clubScore"` // 俱乐部分
	SeatScore      string                          `json:"seatScore"`
	RoundScore     string                          `json:"roundScore"`
	ShouPaiCount   int32                           `json:"shouPaiCount"`
	ChuPai         []int8                          `json:"chuPai"`
	HuPai          []int8                          `json:"huPai"`
	OperationPai   []*gameMaJiang.OperationPaiInfo `json:"operPai"`
	PiaoScore      int64                           `json:"piaoScore"` // 漂分
	LiangDaoPaiArr []int8                          `json:"ldPai"`     // 亮倒牌
	TingPaiArr     []int8                          `json:"ting"`      // 听牌

	VoteStatus int32 `json:"vote"` // 解散桌子 投票
	//OperationTime int64 `json:"operTime"` // 操作剩余时间
}

// 桌子数据
const ID_TableData = 20100

type SC_TableData struct {
	MZCID int32 `json:"mzID"` // 盟主ID
	TableNumber      int32 `json:"tableNum"`    // 房间编号
	TableStatus      int32 `json:"tableStatus"` // 桌子状态
	BankerSeatNumber int32 `json:"banker"`      // 庄家座位号

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
	KouPaiArr          []int8         `json:"kou"`               // 已经扣牌
	ClubScore          string         `json:"clubScore"`         // 俱乐部分
	GameRuleText       string         `json:"gameRule"`          // 游戏规则JSON
	FirstRoundReadTime int64          `json:"FRRT"`              // 首局准备的时间

	DissolveID          int32 `json:"dissolveID"`     // 解散发起人
	LaunchDissolveTime  int64 `json:"dissolveTime"`   // 发起解散时,时间戳
	PlayerOperationTime int64 `json:"playerOperTime"` // 玩家操作时间倒计时
}

// 游戏 小局结束
const ID_RoundGameOver = 20101

type HuPaiSeat struct {
	UID        int64  `json:"uid"`
	NickName   string `json:"nick"`
	Head       string `json:"head"`
	SeatNumber int32  `json:"seatNum"` // 座位号
	//HuMode        int32                           `json:"huMode"`        // 0:没有胡 1:自摸 2:点炮
	HuPai         []uint8                         `json:"huPai"`         // 胡的 牌型 组合
	ShouPai       []int8                          `json:"shouPai"`       // 手牌
	OperationPai  []*gameMaJiang.OperationPaiInfo `json:"operPai"`       // 操作区域的牌
	HuPaiXing     []*gameMaJiang.HuPaiXing        `json:"huFS"`          // 胡的番数
	KouPai        []int8                          `json:"kouPai"`        // 扣牌
	LiangDaoScore string                          `json:"LDScore"`       // 亮倒 输赢分
	IsLiangDao    bool                            `json:"liangDao"`      // 是否亮倒
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

	GangScore   string `json:"gangScore"`
	PiaoScore   string `json:"piaoScore"`
	MaScore     string `json:"maScore"`
	PaiXinScore string `json:"zmordpScore"`
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

// 选漂
const ID_PlayerXuanPiao = 20103

type CS_XuanPiao struct {
	Value int64 `json:"value"`
}

// 通知选漂
const ID_NoticeXuanPiao = 20104

// 广播玩家选漂
const ID_BroadcastPlayerXuanPiao = 20105

type SeatPiaoScore struct {
	SeatNumber int32 `json:"seatNum"`
	Value      int64 `json:"value"`
}
type BroadXuanPiao struct {
	SeatPiaoScoreArr []SeatPiaoScore `json:"seatPiaoScore"`
}

// 广播亮倒
const ID_BroadcastLiangDao = 20106

type CS_BroadcastLiangDao struct {
	SeatNumber int32  `json:"seatNum"`
	PaiArr     []int8 `json:"pai"`  // 亮的牌
	TingPai    []int8 `json:"ting"` // 听牌
}

// 自定义下一张牌
const ID_CustomNextPai = 20107

type CS_CustomNextPai struct {
	Pai int8 `json:"pai"` // 牌
}

// 获取剩余牌的
const ID_GetRemainingPai = 20108

// 亮倒
const ID_LiangDao = 20109

type CS_LiangDao struct {
	LiangDaoPaiArr []int8 `json:"ldPai"`   // 亮的牌
	TingPai        []int8 `json:"ting"`    // 听牌
	KouPai         []int8 `json:"kou"`     // 扣牌()
	PlayPai        int8   `json:"playPai"` // 出的牌
}

type SC_Hu struct {
	Category    int   `json:"category"`
	HuSeat      int32 `json:"huSeat"`
	HuPai       int8  `json:"huPai"`
	PlayPai     int8  `json:"playPai"`
	DianPaoSeat int32 `json:"dpSeat"`
}
