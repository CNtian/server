package gameMaJiang

import (
	"qpGame/qpTable"
)

const (
	Nil = iota // 0
	AnGang
	MingGang
	BuGang
	Nil4
	ZiMo
	DianPao

	HuJiaoZhuanYi // 7
	CaGua         // 8
	XiFen         // 9
	ChaJiaoYing   // 10
	PiaoFen       // 11
	MaFen         // 12
)

type MJSeat struct {
	SeatData *qpTable.SeatData

	GangCount    int32 // 杠牌次数
	DianPaoCount int32 // 点炮次数
	JiePaoCount  int32 // 接炮次数
	HuPaiCount   int32 // 胡牌次数

	//
	IsGuoHu       bool   // 过胡
	GangArr       []int8 // 可以杠的牌
	HuPaiXing     []*HuPaiXing
	HuScore       int64
	LianGangCount int32                                // 连杠的次数
	ShouPai       [MaxHuaSe + 1][MaxDianShu_9 + 1]int8 // 手牌 key:牌  value:数量
	ShouPaiCount  int32                                // 手牌数量
	PlayPai       []int8                               // 出牌
	OperationPai  []*OperationPaiInfo                  // 已经操作过的 牌
	OperationItem PlayerMJOperation                    // 操作项
	CurMoPai      int8                                 // 当前摸的牌

	CustomNextPai int8 // 自定义下一张牌
}

// 清理座位一轮数据
func (this *MJSeat) CleanRoundData() {
	this.IsGuoHu = false
	this.GangArr = nil
	this.HuPaiXing = nil
	this.HuScore = 0
	this.LianGangCount = 0
	this.ShouPai = [MaxHuaSe + 1][MaxDianShu_9 + 1]int8{}
	this.ShouPaiCount = 0
	this.PlayPai = make([]int8, 0, 32)
	this.OperationPai = make([]*OperationPaiInfo, 0, 4)
	this.OperationItem = 0
	this.CurMoPai = InvalidPai
	this.CustomNextPai = InvalidPai

	this.SeatData.CleanRoundData()
}

func (this *MJSeat) GetSeatData() *qpTable.SeatData {
	return this.SeatData
}

func (this *MJSeat) GetXSeatData(int) interface{} {
	return this
}

func (this *MJSeat) PutOperation(value PlayerMJOperation) {

	this.SeatData.MakeOperationID()
	this.OperationItem |= value
}

func (this *MJSeat) SetOperation(value PlayerMJOperation) {

	this.SeatData.MakeOperationID()
	this.OperationItem = value
}

// 添加手牌
func (this *MJSeat) PushShouPai(pai int8) {
	huaSeIndex := uint8(pai) >> 4
	this.ShouPai[huaSeIndex][0] += 1
	this.ShouPai[huaSeIndex][pai&0x0F] += 1
	this.ShouPaiCount += 1
}

// 删除手牌
func (this *MJSeat) DeleteShouPai(pai int8) bool {
	huaSeIndex := uint8(pai) >> 4
	dianShu := uint8(pai) & 0x0F
	if huaSeIndex > uint8(MaxHuaSe) || dianShu > uint8(MaxDianShu_9) {
		return false
	}
	if this.ShouPai[huaSeIndex][dianShu] > 0 {
		this.ShouPai[huaSeIndex][0] -= 1
		this.ShouPai[huaSeIndex][dianShu] -= 1
		this.ShouPaiCount -= 1
		return true
	}
	return false
}

// 牌 是否存在
func (this *MJSeat) GetPaiCount(pai int8) int8 {
	huaSeIndex := uint8(pai) >> 4
	dianShu := uint8(pai) & 0x0F
	if huaSeIndex > uint8(MaxHuaSe) || dianShu > uint8(MaxDianShu_9) {
		return 0
	}
	return this.ShouPai[huaSeIndex][uint8(pai)&0x0F]
}

// 添加操作牌
func (this *MJSeat) PushOperationPai(group *OperationPaiInfo) {
	this.OperationPai = append(this.OperationPai, group)
}

// 查找 碰牌
func (this *MJSeat) FindPengPai(pai int8) *OperationPaiInfo {
	for _, v := range this.OperationPai {
		if v.OperationPXItem == OPX_PENG &&
			pai == v.PaiArr[0] {
			return v
		}
	}
	return nil
}

// 手牌
func (this *MJSeat) GetShouPai() []int8 {

	shouPaiArr := make([]int8, 0, 14)
	for i := MinHuaSe; i <= MaxHuaSe; i++ {
		if this.ShouPai[uint8(i)][0] < 1 {
			continue
		}
		for j := MinDianShu_1; j <= MaxDianShu_9; j++ {
			for k := int8(0); k < this.ShouPai[uint8(i)][j]; k++ {
				shouPaiArr = append(shouPaiArr, (i*0x10)|j)
			}
		}
	}
	return shouPaiArr
}

// 手牌
func (this *MJSeat) RangeShouPai(check func(int8) bool) bool {

	for i := MinHuaSe; i <= MaxHuaSe; i++ {
		if this.ShouPai[uint8(i)][0] < 1 {
			continue
		}
		for j := MinDianShu_1; j <= MaxDianShu_9; j++ {
			for k := int8(0); k < this.ShouPai[uint8(i)][j]; k++ {
				if check(i*0x10+j) == true {
					return false
				}
			}
		}
	}
	return true
}
