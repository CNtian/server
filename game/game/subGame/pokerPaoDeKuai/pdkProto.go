package pokerPDKTable

import "qpGame/qpTable"

type OperationItem int32

// 0:玩家自己操作 1:必须出牌  2:要不起  3:过
const (
	PKOperation         = 0
	PKOperation_PlayPai = 1
	PKOperation_YaoBuQi = 2
	PKOperation_Guo     = 3
)

// 发手牌
const SC_FaShouPai = 3000 // 游戏开始,发手牌

type MsgFaShouPai struct {
	SeatNumber int32  `json:"seatNumber"`
	Pai        []int8 `json:"shouPai"` // (手牌)仅对自己可见
}

// 通知玩家操作
const SC_NoticeOperation = 3001

type MsgNoticeOperation struct {
	SeatNumber    int32         `json:"seatNumber"`
	OperationID   string        `json:"operationID"`
	OperationItem OperationItem `json:"operation"` // 0:玩家自己操作 1:必须出牌  2:要不起  3:过
}

// 广播 当前操作 的 座位号
const B_CurOperationSeatNumber = 3002

type MsgBroadcastOperation struct {
	SeatNumber int32 `json:"seatNumber"`
}

// 玩家出牌
const ID_Play = 3003

type CS_PlayPai struct {
	OperationID string `json:"operationID"`
	Operation   int32  `json:"operation"` // 1:出牌  2:要不起  3:过
	ChuPai      []int8 `json:"pai"`
}

// 广播玩家出牌
const B_PlayerPlay = 3004

type MsgBroadcastPlayerPai struct {
	SeatNum   int32  `json:"seatNum"`
	Operation int32  `json:"operation"` // 1:出牌  2:要不起  3:过
	ChuPai    []int8 `json:"pai"`
	PaiXing   int32  `json:"paiXing"`  // 牌型
	MinValue  int8   `json:"minValue"` // 最小值
}

// 其他玩家信息
type PdkSeatData struct {
	UID               int64  `json:"uid"`
	Nick              string `json:"nick"`
	HeadURL           string `json:"headURL"`
	IP                string `json:"ip"`
	IsGPS             bool   `json:"gps"`
	Sex               int32  `json:"sex"`
	SeatNumber        int32  `json:"seatNum"`    // 座位号
	SeatStatus        uint32 `json:"seatStatus"` // 座位状态
	SeatScore         string `json:"seatScore"`  // 座位游戏分
	ClubID            int32  `json:"cID"`
	ClubScore         string `json:"clubScore"`    // 俱乐部分
	RoundScore        string `json:"roundScore"`   // 当前游戏分
	ShouPaiCount      int32  `json:"shouPaiCount"` // 手牌数量
	CurPlayCard       []int8 `json:"curPlayPai"`   // 最近出的牌
	LastOperationItem int32  `json:"operation"`    // 0:玩家自己操作 1:必须出牌  2:要不起  3:过
	VoteStatus        int32  `json:"vote"`         // 解散桌子 投票
	OperationTime     int64  `json:"operTime"`     // 操作剩余时间
	PaiXing           int32  `json:"paiXing"`      // 牌型
	MinValue          int8   `json:"minValue"`     // 最小值
}

// 请求桌子数据
const CS_TableData = 3005

type MsgTableData struct {
	MZCID int32 `json:"mzID"` // 盟主ID
	TableNumber       int32          `json:"tableNum"`       // 房间编号
	TableStatus       uint32         `json:"tableStatus"`    // 桌子状态
	RoundCount        int32          `json:"curRound"`       // 当前玩局数
	TableRuleText     string         `json:"tableRule"`      // 桌子配置JSON
	CurPlaySeatNumber int32          `json:"curPlaySeatNum"` // 当前出牌的座位号
	NiaoPai           int8           `json:"niaoPai"`        // 鸟牌
	SurplusPai        []int8         `json:"surplusPai"`     // 剩余牌数
	SeatData          []*PdkSeatData `json:"seatData"`       // 座位上的数据
	ShouPai           []int8         `json:"shouPai"`        // 自己的手牌
	OperationID       string         `json:"operationID"`    // 操作ID
	OperationItem     int32          `json:"operation"`      // 0:玩家自己操作 1:必须出牌  2:要不起  3:过
	GameRuleText      string         `json:"gameRule"`       // 游戏规则JSON
	ClubRuleText      string         `json:"clubRule"`       // 俱乐部配置JSON
	ClubScore         string         `json:"clubScore"`      // 俱乐部分

	DissolveID         int32 `json:"dissolveID"`   // 解散发起人
	LaunchDissolveTime int64 `json:"dissolveTime"` // 发起解散时,时间戳
}

// 小局游戏结束
const ID_RoundOver = 3006

type RoundSeatScore struct {
	UID           int64                  `json:"uid"`
	NickName      string                 `json:"nick"`
	Head          string                 `json:"head"`
	Pai           []int8                 `json:"pai"`           //手牌
	BombScore     string                 `json:"bomb"`          // 炸弹分
	IsNiaoPai     bool                   `json:"niaoPai"`       // 是否有鸟牌
	IsChunTian    bool                   `json:"chunTian"`      // 是否春天
	IsFanChun     bool                   `json:"fanChun"`       // 是否反春
	GameScoreStep []qpTable.GameScoreRec `json:"gameScoreStep"` // 游戏中的 得分记录
	RecChuPai     []int8                 `json:"recChuPai"`     // 记录出牌

	GameScore string `json:"gameScore"` // 游戏输赢分
	SeatScore string `json:"seatScore"` // 座位分

	PaiXinScore string `json:"paiScore"`
}
type BroadcastRoundOver struct {
	TableNumber   int32             `json:"tableNum"`   // 房间编号
	SurplusPaiArr []int8            `json:"surplusPai"` // 剩余牌的数量
	SeatData      []*RoundSeatScore `json:"roundSeat"`  // 座位上的数据
	Timestamp     int64             `json:"timestamp"`  // 结束时间
}

const ID_NoticeLuckPai = 3007

type BroadcastLuckPai struct {
	LuckPai int8 `json:"luckPai"` // 幸运牌
	NiaoPai int8 `json:"niaoPai"` // 鸟牌
}

// 大结算
const ID_BroadcastGameOver = 3008

type GameOverSeatData struct {
	ClubID        int32  `json:"clubID"`
	UID           int64  `json:"uid"`
	Nick          string `json:"nick"`
	Head          string `json:"head"`
	ChunTianCount int32  `json:chunTian`   // 春天次数
	BombCount     int32  `json:"bomb"`     // 炸弹次数
	WinCount      int32  `json:"win"`      // 胜利的次数
	LoseCount     int32  `json:"lose"`     // 失败的次数
	SeatScore     string `json:"seaScore"` // 座位分
	SeatScoreInt  int64  `json:"-"`
	IsMaxWin      bool   `json:"isWin"` // 是否大赢家

	BombScore   string `json:"bombScore"`   // 炸弹得分
	PaiXinScore string `json:"PaiXinScore"` // 牌型得分
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
const ID_CustomShouPai = 3009

type CS_CustomShouPai struct {
	ShouPai []int8 `json:"shouPai"`
}
type SC_CustomShouPai struct {
	ShouPai []int8 `json:"shouPai"`
}
