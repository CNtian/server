package gameMaJiang

// 胡牌的牌型
type HuPaiXing struct {
	PaiXing int32 `json:"px"`
	FanShu  int64 `json:"fs"`
}

// 玩家操作
type PlayerMJOperation uint32

const (
	OPI_CHI     PlayerMJOperation = 1
	OPI_PENG    PlayerMJOperation = 2
	OPI_GANG    PlayerMJOperation = 4
	OPI_HU      PlayerMJOperation = 8
	OPI_PlayPai PlayerMJOperation = 16
	OPI_MO_Pai  PlayerMJOperation = 32
)

// 操作项
type OperationPaiXingItem uint32

const (
	OPX_CHI       OperationPaiXingItem = 1
	OPX_PENG      OperationPaiXingItem = 2
	OPX_MING_GANG OperationPaiXingItem = 4
	OPX_BU_GANG   OperationPaiXingItem = 8
	OPX_AN_GANG   OperationPaiXingItem = 16
)

// 已操作牌的牌型
type OperationPaiInfo struct {
	OperationPXItem   OperationPaiXingItem `json:"px"`             // 牌型
	PlayPaiSeatNumber int32                `json:"playSeatNumber"` // 出牌座位号
	Name              string               `json:"name"`
	PaiArr            []int8               `json:"pai"` // 牌
}

// 游戏开始,发手牌
const ID_FaShouPai = 2000

type SC_FaShouPai struct {
	SeatNum       int32  `json:"seatNum"`
	Pai           []int8 `json:"pai"`    // (手牌)仅对自己可见
	BankerSeatNum int32  `json:"banker"` // 庄家座位号
}

// 玩家过操作(针对吃碰杠胡)
const ID_Guo = 2001

type CS_Guo struct {
	OperationID string `json:"operationID"`
}

// 玩家出牌
const ID_Play = 2002

type CS_Play struct {
	OperationID string `json:"operationID"`
	Pai         int8   `json:"pai"`
}

// 玩家吃牌
const ID_Chi = 2003

type CS_Chi struct {
	OperationID string `json:"operationID"`
	Pai         []int8 `json:"pai"` // 必须是 3 张
}

// 玩家碰牌
const ID_Peng = 2004

type CS_Peng struct {
	OperationID string `json:"operationID"`
	Pai         int8   `json:"pai"`
}

// 玩家杠牌
const ID_Gang = 2005

type CS_Gang struct {
	OperationID string `json:"operationID"`
	Pai         int8   `json:"pai"`
}

// 玩家胡牌
const ID_Hu = 2006

type CS_Hu struct {
	OperationID string `json:"operationID"`
}

// 通知玩家操作
const ID_NoticeOperation = 2007

type SC_NoticeOperation struct {
	SeatNumber  int32             `json:"seatNum"`
	OperationID string            `json:"operationID"`
	Operation   PlayerMJOperation `json:"operItem"`
	Pai         int8              `json:"pai"`
	GangPai     []int8            `json:"gang"` // 可杠的牌
}

// 玩家摸牌
const ID_PlayerMoPai = 2008

type SC_PlayerMoPai struct {
	Card        int8              `json:"pai"` // (手牌)仅对自己可见
	OperationID string            `json:"operationID"`
	Operation   PlayerMJOperation `json:"operItem"`
	GangArr     []int8            `json:"gang"` // 可杠的牌
}

// 广播玩家出牌
const ID_BroadcastPlay = 2009

type BroadcastPlay struct {
	SeatNumber int32 `json:"seatNum"`
	Pai        int8  `json:"pai"`
}

// 广播玩家吃牌
const ID_BroadcastChi = 2010

type BroadcastChi struct {
	SeatNumber  int32  `json:"seatNum"`
	Pai         []int8 `json:"pai"`         // 必须是 3 张
	ChiPai      int8   `json:"chiPai"`      // 吃的牌
	PlaySeatNum int32  `json:"playSeatNum"` // 谁出的牌
}

// 广播玩家碰牌
const ID_BroadcastPeng = 2011

type BroadcastPeng struct {
	SeatNumber  int32 `json:"seatNum"`
	Pai         int8  `json:"pai"`
	PlaySeatNum int32 `json:"playSeatNum"` // 谁出的牌
}

// 广播玩家杆牌
const ID_BroadcastGang = 2012

type BroadcastGang struct {
	SeatNumber  int32 `json:"seatNum"`
	Pai         int8  `json:"pai"`         // 暗杠,此字段无效
	Type        int32 `json:"type"`        // 1: 手上3张,别人 打出 1张  2: 自己4张  3： 碰了一次, 自摸1张
	PlaySeatNum int32 `json:"playSeatNum"` // 谁出的牌(暗杠,此字段无效)
}

// 广播玩家胡牌
//const ID_BroadcastHu = 2013
//
//type BroadcastHu struct {
//	SeatNumber int32 `json:"seatNum"`
//}

// 广播玩家摸牌
const ID_BroadcastMoPai = 2014

type BroadcastMoPai struct {
	SeatNumber int32 `json:"seatNum"`
	CardCount  int32 `json:"cardCount"` // 剩余牌数
}

// 明杠信息
type ProMingGang struct {
	Pai        int8  `json:"pai"`
	SeatNumber int32 `json:"seatNum"` //明杠谁的牌(座位号)
}
