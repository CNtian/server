package ZhaJinHua

// 请求桌子数据
const ID_TableData = 3200

type ZhaJinHuaSeatData struct {
	UID         int64   `json:"uid"`
	Nick        string  `json:"nick"`
	HeadURL     string  `json:"headURL"`
	IP          string  `json:"ip"`
	IsGPS       bool    `json:"gps"`
	Sex         int32   `json:"sex"`
	SeatNumber  int32   `json:"seatNum"`    // 座位号
	SeatStatus  uint32  `json:"seatStatus"` // 座位状态
	SeatScore   string  `json:"seatScore"`  // 座位游戏分
	ClubID      int32   `json:"cID"`
	ClubScore   string  `json:"clubScore"`  // 俱乐部分
	RoundScore  string  `json:"roundScore"` // 当前游戏分
	ShouPai     []int8  `json:"shouPai"`    // 手牌
	XiaZhuScore float64 `json:"xiaZhu"`     // 下注
	IsQiPai     bool    `json:"qiPai"`      // 弃牌
	IsLose      bool    `json:"losed"`      // 比牌输了
	IsKanPai    bool    `json:"kanPai"`     // 是否看牌
	XiaZhuTime  int32   `json:"xzTime"`     // 下注次数

	VoteStatus    int32 `json:"vote"`     // 解散桌子 投票
	OperationTime int64 `json:"operTime"` // 操作剩余时间
}
type SC_TableData struct {
	TableNumber   int32  `json:"tableNum"`    // 房间编号
	TableStatus   uint32 `json:"tableStatus"` // 桌子状态
	MZCID         int32  `json:"mzID"`        // 盟主ID
	RoundCount    int32  `json:"curRound"`    // 当前玩局数
	TableRuleText string `json:"tableRule"`   // 桌子配置JSON
	//SurplusPai         []int8               `json:"surplusPai"`  // 剩余牌数
	BankerSeatNum      int32                `json:"banker"`  // 庄家座位号
	XiaZhuRound        int32                `json:"xzRound"` // 下注🎡轮数
	CurSeatNumber      int32                `json:"curSeat"`
	MaxXiaZhuCount     int32                `json:"curMaxXZ"` // 最大下注
	SeatData           []*ZhaJinHuaSeatData `json:"seatData"` // 座位上的数据
	ShouPai            []int8               `json:"shouPai"`  // 自己的手牌
	ClubID             int32                `json:"clubID"`
	IsGenDaoDi         bool                 `json:"genDaoDi"` // 跟到底
	PaiXing            int32                `json:"paiXing"`  // 牌型
	FirstRoundReadTime int64                `json:"FRRT"`     // 首局准备的时间

	GameRuleText string `json:"gameRule"`  // 游戏规则JSON
	ClubRuleText string `json:"clubRule"`  // 俱乐部配置JSON
	ClubScore    string `json:"clubScore"` // 俱乐部分

	DissolveID         int32 `json:"dissolveID"`   // 解散发起人
	LaunchDissolveTime int64 `json:"dissolveTime"` // 发起解散时,时间戳
}

// 测试手牌
const ID_CustomShouPai = 3201

type CS_CustomShouPai struct {
	ShouPai []int8 `json:"shouPai"`
}

// 发手牌
const SC_FaShouPai = 3202

type MsgGameStart struct {
	SeatNumber []int32 `json:"seat"` // 有牌的座位号
}

// 小局游戏结束
const ID_RoundOver = 3204

type RoundSeatScore struct {
	ClubID     int32  `json:"clubID"`
	UID        int64  `json:"uid"`
	NickName   string `json:"nick"`
	Head       string `json:"head"`
	SeatNumber int32  `json:"seatNum"` // 座位号
	Pai        []int8 `json:"pai"`     //手牌
	PaiXing    int32  `json:"paiXing"`
	IsQiPai    bool   `json:"qiPai"` // 是否弃牌

	GameScore string `json:"gameScore"` // 游戏输赢分
	SeatScore string `json:"seatScore"` // 座位分
}
type BroadcastRoundOver struct {
	TableNumber int32 `json:"tableNum"` // 房间编号
	//SurplusPaiArr []int8            `json:"surplusPai"` // 剩余牌的数量
	SeatData  []*RoundSeatScore `json:"roundSeat"` // 座位上的数据
	Timestamp int64             `json:"timestamp"` // 结束时间

	ClubID     int32 `json:"clubID"`     // 盟主圈子ID
	ClubPlayID int64 `json:"clubPlayID"` // 盟主玩法ID
}

// 大结算
const ID_BroadcastGameOver = 3205

type GameOverSeatData struct {
	ClubID       int32  `json:"clubID"`
	UID          int64  `json:"uid"`
	Nick         string `json:"nick"`
	Head         string `json:"head"`
	MaxPaiXing   int32  `json:"maxPX"`    // 最大牌型
	MaxGetScore  string `json:"maxGS"`    // 最大得分
	WinCount     int32  `json:"win"`      // 胜利的次数
	LoseCount    int32  `json:"lose"`     // 失败的次数
	SeatScore    string `json:"seaScore"` // 座位分
	SeatScoreInt int64  `json:"-"`
	IsMaxWin     bool   `json:"isWin"` // 是否大赢家
}

type BroadcastGameOver struct {
	TableNumber  int32 `json:"tableNum"`
	CurRound     int32 `json:"curRound"`   // 当前局数
	MaxRound     int32 `json:"maxRound"`   // 总局数
	EndTime      int64 `json:"endTime"`    // 结束时间
	DissolveType int32 `json:"dissolve"`   // 解散类型
	ClubID       int32 `json:"clubID"`     // 盟主圈子ID
	ClubPlayID   int64 `json:"clubPlayID"` // 盟主玩法ID

	SeatData NNGameOverSeat `json:"seat"` // 座位信息
}

type NNGameOverSeat []*GameOverSeatData

func (s NNGameOverSeat) Len() int      { return len(s) }
func (s NNGameOverSeat) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s NNGameOverSeat) Less(i, j int) bool {
	return s[i].SeatScoreInt > s[j].SeatScoreInt
}

// 下注
const ID_XiaZhu = 3206

type CS_XiaZhu struct {
	XiaZhu float64 `json:"xiaZhu"`
}

// 广播下注
const ID_BroadcastXiaZhu = 3207

type SC_XiaZhu struct {
	SeatNumber  int32   `json:"seat"`
	LeaveXiaZhu int32   `json:"levelXZ"`
	IndexXiaZhu int32   `json:"indexXZ"`
	XiaZhu      float64 `json:"xiaZhu"`
	IsJiaZhu    bool    `json:"isJiaZhu"`
	XiaZhuCount float64 `json:"xzCount"` // 玩家下注总数
}

// 通知下 底注
const ID_NoticeXiaDiZhu = 3208

type SC_NoticeXiaDiZhu struct {
	BankerSeatNum int32   `json:"banker"`
	DiZhu         float64 `json:"diZhu"` // 底注
}

// 看牌
const ID_KanPai = 3209

type SC_KanPai struct {
	PaiArr  []int8 `json:"pai"`
	PaiXing int32  `json:"px"`
}

// 广播看牌
const ID_BroadcastKanPai = 3210

type SC_BroadcastKanPai struct {
	SeatNumber int32 `json:"seatNum"`
}

// 通知操作
const ID_NoticeOperation = 3212

type CS_NoticeOperation struct {
	TargetSeat int32 `json:"target"` // 目标座位号
}

// 弃牌
const ID_QiPai = 3213

// 广播弃牌
const ID_BroadcastQiPai = 3213

type SC_BroadcastQiPai struct {
	SeatNumber int32 `json:"seatNum"`
}

// 比牌
const ID_BiPai = 3214

type CS_BiPai struct {
	TargetSeat int `json:"target"` // 目标座位号
}

// 广播比牌结果
const ID_BroadcastBiPai = 3215

type SC_BroadcastBiPai struct {
	InitiatorSeat int32 `json:"initiator"`
	WinSeat       int32 `json:"win"`  // 赢 座位号
	LoseSeat      int32 `json:"lose"` // 输 座位号
}

// 跟到底
const ID_GenDaoDi = 3216

type CS_GenDaoDi struct {
	On bool `json:"on"`
}

// 下注轮数 变化
const ID_XiaZhuRoundChanged = 3217

type SC_XiaZhuRoundChanged struct {
	XiaZhuRound int32 `json:"xzRound"`
}

const ID_PlayBack = 3218

type SC_PlayBackFaShouPai struct {
	SeatNumber int32  `json:"seat"`
	Pai        []int8 `json:"shouPai"` // (手牌)仅对自己可见
}

const ID_GetPai = 3219

type GetPai struct {
	UID    int64  `json:"uid"`
	SeatNo int32  `json:"seatNo"`
	Pai    []int8 `json:"pai"`
}
type SC_GetPai struct {
	Pai []GetPai `json:"pPai"`
}

// 换牌
const ID_ChangePai = 3220

type CS_ChangePai struct {
	Pro int `json:"pro"`
}
