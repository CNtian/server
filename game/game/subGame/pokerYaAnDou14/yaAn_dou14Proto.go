package pokerYaAnD14

import "qpGame/qpTable"

// 玩家操作
type D14Operation uint32

const (
	OPI_CHI     D14Operation = 1
	OPI_PENG    D14Operation = 2
	OPI_GANG    D14Operation = 4
	OPI_HU      D14Operation = 8
	OPI_PlayPai D14Operation = 16
	OPI_MO_Pai  D14Operation = 32
	OPI_TouMo   D14Operation = 64
	OPI_Bao     D14Operation = 128
)

// 发手牌
const SC_FaShouPai = 4000 // 游戏开始,发手牌

type MsgFaShouPai struct {
	SeatNumber int32  `json:"seatNo"`
	Pai        []int8 `json:"shouPai"` // (手牌)仅对自己可见
}

// 通知玩家操作
const SC_NoticeOperation = 4001

type MsgNoticeOperation struct {
	OperationID   string       `json:"operID"`
	OperationItem D14Operation `json:"operItem"`
	AllCanGangPai []int8       `json:"gang"`

	IsPlay int8 `json:"chuPai"` // 针对出牌
	IsFan  int8 `json:"fanPai"` // 针对翻牌
}

// 广播 当前操作 的 座位号
const B_CurOperationSeatNumber = 4002

type MsgBroadcastOperation struct {
	SeatNumber qpTable.SeatNumber `json:"seatNo"`
}

// 玩家出牌
const ID_Play = 4003

type CS_PlayPai struct {
	OperationID string `json:"operID"`
	ChuPai      int8   `json:"pai"`
}

// 广播玩家出牌
const B_PlayerPlay = 4004

type MsgBroadcastPlayerPai struct {
	SeatNum int32 `json:"seatNo"`
	ChuPai  int8  `json:"pai"`
}

// 其他玩家信息
type PdkSeatData struct {
	UID           int64  `json:"uid"`
	Nick          string `json:"nick"`
	HeadURL       string `json:"headURL"`
	IP            string `json:"ip"`
	IsGPS         bool   `json:"gps"`
	Sex           int32  `json:"sex"`
	SeatNumber    int32  `json:"seatNo"`     // 座位号
	SeatStatus    uint32 `json:"seatStatus"` // 座位状态
	SeatScore     string `json:"seatScore"`  // 座位游戏分
	ClubID        int32  `json:"cID"`
	ClubScore     string `json:"clubScore"`  // 俱乐部分
	RoundScore    string `json:"roundScore"` // 当前游戏分
	ShouPaiCount  int    `json:"spCount"`    // 手牌数量
	VoteStatus    int32  `json:"vote"`       // 解散桌子 投票
	OperationTime int64  `json:"operTime"`   // 操作剩余时间

	ChiPai   []SortPai `json:"chiP"`    // 吃牌
	PengPai  []SortPai `json:"pengP"`   // 碰牌
	GangPai  []SortPai `json:"gangP"`   // 杠牌
	PlayCard []int8    `json:"playPai"` // 出牌
	TouPai   []SortPai `json:"touPai"`  // 偷牌
}

// 请求桌子数据
const CS_TableData = 4005

type MsgTableData struct {
	MZCID         int32  `json:"mzID"`        // 盟主ID
	TableNumber   int32  `json:"tableNum"`    // 房间编号
	TableStatus   uint32 `json:"tableStatus"` // 桌子状态
	RoundCount    int32  `json:"curRound"`    // 当前玩局数
	TableRuleText string `json:"tableRule"`   // 桌子配置JSON
	ClubRuleText  string `json:"clubRule"`    // 俱乐部配置JSON
	ClubScore     string `json:"clubScore"`   // 俱乐部分

	BankerNo         int32 `json:"bankerNo"`  // 庄家 座位号
	XiaoJiaNo        int32 `json:"xiaoNo"`    // 庄家 座位号
	MoPaiNo          int32 `json:"moPaiNo"`   // 摸牌 座位号
	ChuPaiNo         int32 `json:"playPaiNo"` // 出牌 座位号
	ChuPai           int8  `json:"chuPai"`    // 出牌
	FanPaiNo         int32 `json:"fanPaiNo"`  // 翻牌 座位号
	FanPai           int8  `json:"fanPai"`    // 翻的牌
	CurPointToSeatNo int32 `json:"curNo"`     // 报牌座位号

	SurplusPaiCount int32          `json:"surplusPai"` // 剩余牌
	SeatData        []*PdkSeatData `json:"seatData"`   // 座位上的数据
	ShouPai         []int8         `json:"shouPai"`    // 自己的手牌
	CanGang         []int8         `json:"gang"`       // 可杠的牌
	OperationID     string         `json:"operID"`     // 操作ID
	OperationItem   int32          `json:"operItem"`   // 0:玩家自己操作 1:必须出牌  2:要不起  3:过
	GameRuleText    string         `json:"gameRule"`   // 游戏规则JSON

	DissolveID          int32 `json:"dissolveID"`   // 解散发起人
	LaunchDissolveTime  int64 `json:"dissolveTime"` // 发起解散时,时间戳
	PlayerOperationTime int64 `json:"operTime"`     // 玩家操作时间倒计时
}

// 小局游戏结束
const ID_RoundOver = 4006

type RoundSeatScore struct {
	UID        int64  `json:"uid"`
	NickName   string `json:"nick"`
	Head       string `json:"head"`
	SeatNumber int32  `json:"seatNum"` // 座位号

	//XiaoJia  bool      `json:"xiaoJia"` // 小家
	ChiPai   []SortPai `json:"chiP"`    // 吃牌
	PengPai  []SortPai `json:"pengP"`   // 碰牌
	GangPai  []SortPai `json:"gangP"`   // 杠牌
	PlayCard []int8    `json:"playPai"` // 出牌
	TouPai   []SortPai `json:"touPai"`  // 偷牌
	ShouPai  []int8    `json:"shouPai"` // 手牌
	//DiFen    int       `json:"diFen"`    // 底分
	WinScore int `json:"winScore"` // 底分*牌型翻 赢分

	GameScoreStep []qpTable.GameScoreRec `json:"gameScoreStep"` // 游戏中的 得分记录

	GameScore string `json:"gameScore"` // 游戏输赢分
	SeatScore string `json:"seatScore"` // 座位分
}
type BroadcastRoundOver struct {
	TableNumber   int32              `json:"tableNum"`   // 房间编号
	SurplusPaiArr []int8             `json:"surplusPai"` // 剩余的牌
	DianPaoSeatNo qpTable.SeatNumber `json:"dpNo"`       // 点炮的座位号
	HuSeatNo      int32              `json:"huSeatNo"`   // 胡的座位号
	HuPai         int8               `json:"huPai"`      // 胡的牌

	SeatData  []*RoundSeatScore `json:"roundSeat"` // 座位上的数据
	Timestamp int64             `json:"timestamp"` // 结束时间
}

// 大结算
const ID_BroadcastGameOver = 4008

type GameOverSeatData struct {
	ClubID int32  `json:"clubID"`
	UID    int64  `json:"uid"`
	Nick   string `json:"nick"`
	Head   string `json:"head"`

	HupaiCount   int    `json:"hpC"`
	DianPaoCount int    `json:"dpC"`
	ZiMoCount    int    `json:"zmC"`
	SeatScore    string `json:"seaScore"` // 座位分
	SeatScoreInt int64  `json:"-"`        // 判断大赢家时 使用
	IsMaxWin     bool   `json:"isWin"`    // 是否大赢家
}

type BroadcastGameOver struct {
	TableNumber  int32 `json:"tableNum"`
	CurRound     int32 `json:"curRound"`   // 当前局数
	MaxRound     int32 `json:"maxRound"`   // 总局数
	EndTime      int64 `json:"endTime"`    // 结束时间
	DissolveType int32 `json:"dissolve"`   // 解散类型
	ClubID       int32 `json:"clubID"`     // 盟主圈子ID
	ClubPlayID   int64 `json:"clubPlayID"` // 盟主玩法ID

	SeatData PdkGameOverSeat `json:"seat"` // 座位信息
}

type PdkGameOverSeat []*GameOverSeatData

func (s PdkGameOverSeat) Len() int      { return len(s) }
func (s PdkGameOverSeat) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s PdkGameOverSeat) Less(i, j int) bool {
	return s[i].SeatScoreInt > s[j].SeatScoreInt
}

// 自定义手牌
const ID_CustomShouPai = 4009

type CS_CustomShouPai struct {
	ShouPai []int8 `json:"shouPai"`
}
type SC_CustomShouPai struct {
	ShouPai []int8 `json:"shouPai"`
}

// 通知手牌信息
const ID_NoticeShouPaiInfo = 4010

type SC_NoticeShouPaiInfo struct {
	SeatShouPaiCount []int32 `json:"shouPaiC"`
	BankerSeat       int32   `json:"banker"`
	XiaoSeat         int32   `json:"xiao"`
	SurplusPai       int32   `json:"surplusPai"`
}

// 玩家偷摸
const ID_TouMo = 4012

type CS_TouMo struct {
	OperationID string  `json:"operID"`
	Tou         SortPai `json:"tou"`
}

// 玩家偷
const ID_BroadcastTouMo = 4013

type SC_BroadcastTouMo struct {
	SeatNo int32  `json:"sNo"`
	Tou    []int8 `json:"tou"`
	IsFan  int8   `json:"fan"`
}

// 玩家报
const ID_Bao = 4014

type CS_Bao struct {
	OperationID string `json:"operID"`
}

// 玩家报
const ID_BroadcastTouBao = 4015

type SC_BroadcastTouBao struct {
	SeatNo int32 `json:"seatNo"`
}

// 通知 偷后 摸的牌
const ID_NoticeMoPai = 4016

type SC_NoticeMoPai struct {
	Pai []int8 `json:"pai"`
}

// 广播 翻牌
const ID_BroadcastFanPai = 4017

type SC_BroadcastFanPai struct {
	SeatNo     int32 `json:"seatNo"`
	Pai        int8  `json:"pai"`
	SurplusPai int32 `json:"surplusPai"`
}

// 玩家杠
const ID_Gang = 4018

type CS_Gang struct {
	OperationID string `json:"operID"`
	PaiArr      []int8 `json:"pai"`
}

// 广播杠
const ID_BroadcastGang = 4019

type SC_BroadcastGang struct {
	SeatNo   int32  `json:"seatNo"`
	PaiArr   []int8 `json:"pai"`
	GangType int    `json:"category"`
	AnTouPai int8   `json:"anTou"`
}

// 玩家碰
const ID_Peng = 4020

type CS_Peng struct {
	OperationID string `json:"operID"`
	PaiArr      []int8 `json:"pai"`
}

// 广播碰
const ID_BroadcastPeng = 4021

type SC_BroadcastPeng struct {
	SeatNo int32  `json:"seatNo"`
	PaiArr []int8 `json:"pai"`
}

// 玩家吃
const ID_Chi = 4022

type CS_Chi struct {
	OperationID string `json:"operID"`
	Pai         int8   `json:"pai"`
}

// 广播吃
const ID_BroadcastChi = 4023

type SC_BroadcastChi struct {
	SeatNo int32  `json:"seatNo"`
	Pai    []int8 `json:"pai"`
}

// 玩家胡
const ID_Hu = 4024

type CS_Hu struct {
	OperationID string `json:"operID"`
}

// 广播胡
const ID_BroadcastHu = 4025

type SC_BroadcastHu struct {
	SeatNo int32 `json:"seatNo"`
	Pai    int8  `json:"pai"`
}

// 过
const ID_Guo = 4026

type CS_Guo struct {
	OperationID string `json:"operID"`
}

// 广播摸牌
const ID_BroadcastMoPai = 4027

type SC_BroadcastMoPai struct {
	SeatNo     int32 `json:"seatNo"`
	PaiC       int   `json:"paiC"`
	SurplusPai int32 `json:"surplusPai"`
}

// 测试摸牌
const ID_CustomMoPai = 4028

type CS_CustomMoPai struct {
	Pai int8 `json:"pai"`
}

// 获取剩余的牌
const ID_GetSurplus = 4029

// 翻牌 放入 出牌区
const ID_FanPaiPutPlay = 4030

type SC_BroadcastFanPaiPutPlay struct {
	SeatNo int32 `json:"hello"`
}

// 报 阶段  过
const ID_Bao_Guo = 4031

type SC_BroadcastBao_Guo struct {
	SeatNo qpTable.SeatNumber `json:"seatNo"`
}

// 暗偷
const ID_NoticeAnTou = 4032
