package NiuNiu_mpQiangZhuang

// 请求桌子数据
const ID_TableData = 3100

type NiuNiuSeatData struct {
	UID        int64  `json:"uid"`
	Nick       string `json:"nick"`
	HeadURL    string `json:"headURL"`
	IP         string `json:"ip"`
	IsGPS      bool   `json:"gps"`
	Sex        int32  `json:"sex"`
	SeatNumber int32  `json:"seatNum"`    // 座位号
	SeatStatus uint32 `json:"seatStatus"` // 座位状态
	SeatScore  string `json:"seatScore"`  // 座位游戏分
	ClubID     int32  `json:"cID"`
	ClubScore  string `json:"clubScore"`  // 俱乐部分
	RoundScore string `json:"roundScore"` // 当前游戏分
	PaiXing    int32  `json:"paiXing"`    // 牌型
	ShouPai    []int8 `json:"shouPai"`    // 手牌
	XiaZhu     int32  `json:"xiaZhu"`     // 下注
	QZValue    int32  `json:"qz"`         // 抢庄
	Liang      bool   `json:"liang"`      // 亮
	LastPai    int8   `json:"lastPai"`    // 最后一张牌

	VoteStatus    int32 `json:"vote"`     // 解散桌子 投票
	OperationTime int64 `json:"operTime"` // 操作剩余时间
}
type SC_TableData struct {
	TableNumber   int32  `json:"tableNum"`    // 房间编号
	TableStatus   uint32 `json:"tableStatus"` // 桌子状态
	MZCID         int32  `json:"mzID"`        // 盟主ID
	RoundCount    int32  `json:"curRound"`    // 当前玩局数
	TableRuleText string `json:"tableRule"`   // 桌子配置JSON
	//SurplusPai         []int8            `json:"surplusPai"`  // 剩余牌数
	BankerSeatNum      int32             `json:"banker"`   // 庄家座位号
	SeatData           []*NiuNiuSeatData `json:"seatData"` // 座位上的数据
	ShouPai            []int8            `json:"shouPai"`  // 自己的手牌
	ClubID             int32             `json:"clubID"`
	GameRuleText       string            `json:"gameRule"`  // 游戏规则JSON
	ClubRuleText       string            `json:"clubRule"`  // 俱乐部配置JSON
	ClubScore          string            `json:"clubScore"` // 俱乐部分
	FirstRoundReadTime int64             `json:"FRRT"`      // 首局准备的时间
	LaiZiPai           int8              `json:"lzPai"`     // 赖子牌

	StageTime     int64   `json:"stageT"` // 阶段时间
	TuiZhuSeatArr []int32 `json:"tzSeat"` // 推注
	TuiZhuArr     []int32 `json:"tz"`     // 推注

	DissolveID         int32 `json:"dissolveID"`   // 解散发起人
	LaunchDissolveTime int64 `json:"dissolveTime"` // 发起解散时,时间戳
}

// 测试手牌
const ID_CustomShouPai = 20107

type CS_CustomShouPai struct {
	ShouPai int8 `json:"pai"`
}

// 开始抢庄
//const ID_StartToQiangZhuang = 3105

//type SC_StartToQiangZhuang struct {
//	PlayingSeatArr []int32 `json:"seat"`
//}

// 玩家抢庄
const ID_PlayerQiangZhuang = 3106

type CS_QiangZhuang struct {
	Value int32 `json:"value"` // -1:未操作 0：不抢  1 2 3 4
}

// 广播玩家抢庄
const ID_BroacastQiangZhuang = 3107

type CS_BroacastQiangZhuang struct {
	SeatNum int32 `json:"seatNum"`
	Value   int32 `json:"value"`
}

// 通知庄家
const ID_NoticeZhuangJia = 3108

type SC_ZhuangJia struct {
	SeatNumber       int32   `json:"setNum"`
	Value            int32   `json:"value"`
	MaxXiaZhuSeatArr []int32 `json:"seat"`
}

// 发手牌
const ID_FaShouPai = 3111

type SC_FaShouPai struct {
	SeatNumber  int32   `json:"seatNumber"`
	Pai         []int8  `json:"shouPai"` // (手牌)仅对自己可见
	PlayingSeat []int32 `json:"seat"`

	LaiZiPai int8 `json:"lzPai"` // 赖子牌
}

// 通知下注
const ID_NoticeXiaZhu = 3112

type SC_NoticeXiaZhu struct {
	PlayingSeatArr []int32 `json:"seat"`
	TuiZhuSeatArr  []int32 `json:"tzSeat"`
	TuiZhu         []int32 `json:"tz"` //5 10  / 30 50  / 150 200
}

// 玩家下注
const ID_XiaZhu = 3113

type CS_XiaZhu struct {
	Value int32 `json:"value"`
}

// 广播玩家下注
const ID_BroadcastXiaZhu = 3114

type SC_XiaZhu struct {
	SeatNumber int32 `json:"seatNumber"`
	XiaZhu     int32 `json:"xiaZhu"`
}

// 亮牌
const ID_PlayerLiangPai = 3115

// 广播 单个玩家 亮牌
const ID_BroadcastLiangPai = 3116

type SC_BroadcastLiangPai struct {
	LiangPaiXing
}

// 广播 所有玩家牌型
const ID_BroadcastLiangPaiXing = 3117

type LiangPaiXing struct {
	SeatNumber   int32  `json:"seatNum"`
	PaiArr       []int8 `json:"pai"` // 3 2:按3 2的顺序排好
	PaiXing      int32  `json:"paiXing"`
	LastPai      int8   `json:"lastPai"` // 最后一张牌
	LaiZiChanged []int8 `json:"lzCom"`   // 赖子变成了什么
}
type SS_LiangPaiXing struct {
	PaiXingArr []LiangPaiXing `json:"paiXing"`
}

// 小局游戏结束
const ID_RoundOver = 3120

type RoundSeatScore struct {
	ClubID     int32  `json:"clubID"`
	UID        int64  `json:"uid"`
	NickName   string `json:"nick"`
	Head       string `json:"head"`
	SeatNumber int32  `json:"seatNum"` // 座位号
	Pai        []int8 `json:"pai"`     //手牌
	LaiZi      []int8 `json:"lz"`      // 赖子变成的牌

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

// 获取剩余牌的
const ID_GetRemainingPai = 20108

// 倒计时
//const ID_StartCountdownClock = 3122

//// 大结算
const ID_BroadcastGameOver = 3121

type GameOverSeatData struct {
	ClubID           int32  `json:"clubID"`
	UID              int64  `json:"uid"`
	Nick             string `json:"nick"`
	Head             string `json:"head"`
	TuiZhuCount      int32  `json:"tzCount"`  // 推注次数
	MaxPaiXing       int32  `json:"maxPX"`    // 最大牌型
	QiangZhuangCount int32  `json:"qzCount"`  // 抢庄次数
	SeatScore        string `json:"seaScore"` // 座位分
	SeatScoreInt     int64  `json:"-"`
	IsMaxWin         bool   `json:"isWin"` // 是否大赢家
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

const ID_GetPai = 3122

type GetPai struct {
	UID int64  `json:"uid"`
	Pai []int8 `json:"pai"`
}
type SC_GetPai struct {
	Pai []GetPai `json:"pPai"`
}

// 换牌
const ID_ChangePai = 3123

type CS_ChangePai struct {
	Pro int `json:"pro"`
}
