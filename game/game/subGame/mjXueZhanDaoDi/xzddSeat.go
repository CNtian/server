package mjXZDDTable

import (
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
)

const SS_HU = qpTable.SS_CustomDefineBase //自定义状态起始值

type XZDDSeat struct {
	MJSeat *gameMaJiang.MJSeat

	HuMode         int32              // 胡的方式
	HuPai          int8               // 胡的牌
	HuOrder        int32              // 胡的顺序
	DGHPlayPaiSeat qpTable.SeatNumber // 点杠花 出牌座位号
	//ZhuanYiGangScore float64            // 转移的获得杠分
	PlayPaiCount  int32              // 出牌次数
	MoPaiCount    int32              // 摸牌次数
	HuFanShu      int64              // 番数
	FanXingMap    map[int32]int64    // 番型
	LastGangScore float64            //上次杠分
	IsGuoHu       bool               // 过胡
	GuoHuFengDing int64              // 过胡封顶
	isZiMo        bool               // 是否是自摸
	GangScore     float64            // 杠
	DingQue       int8               // 定缺
	ChangePai     []int8             // 换牌(自己选的)
	ChangedPai    []int8             // 换牌(换过之后的)
	WinMap        map[int32]struct{} // 赢了哪些座位

	ReadyChangePai []int8
	ReadyDingQue   int8
}

// 清理座位一轮数据
func (this *XZDDSeat) CleanRoundData() {
	this.HuMode = -1
	this.HuPai = gameMaJiang.InvalidPai
	this.HuOrder = -1
	this.DGHPlayPaiSeat = qpTable.INVALID_SEAT_NUMBER
	//this.ZhuanYiGangScore = 0
	this.PlayPaiCount = 0
	this.MoPaiCount = 0
	this.HuFanShu = 0
	this.FanXingMap = nil
	this.LastGangScore = 0
	this.IsGuoHu = false
	this.GuoHuFengDing = 0
	this.isZiMo = false
	this.DingQue = -1
	this.GangScore = 0
	this.ChangePai = nil
	this.ChangedPai = nil
	this.WinMap = nil
	this.MJSeat.SeatData.DelState(SS_HU)
	this.MJSeat.CleanRoundData()
}

func (this *XZDDSeat) GetSeatData() *qpTable.SeatData {
	return this.MJSeat.GetSeatData()
}

func (this *XZDDSeat) GetXSeatData(value int) interface{} {
	if value == 0 {
		return this.MJSeat
	}
	return nil
}

func (this *XZDDSeat) GetGroupPai(count int) []int8 {
	groupArr := make([]int8, 0, count)
	for i := gameMaJiang.MinHuaSe; i <= gameMaJiang.MaxHuaSe; i++ {
		if this.MJSeat.ShouPai[uint8(i)][0] < 1 {
			continue
		}
		for j := gameMaJiang.MinDianShu_1; j <= gameMaJiang.MaxDianShu_9; j++ {
			for k := int8(0); k < this.MJSeat.ShouPai[uint8(i)][j]; k++ {
				groupArr = append(groupArr, (i*0x10)|j)
				if len(groupArr) >= count {
					return groupArr
				}
			}
		}
	}
	return groupArr
}

func (this *XZDDSeat) GetAutoHuanPai(count int) []int8 {
	groupArr := make([]int8, 0, count)

	_minHuSe := gameMaJiang.MinHuaSe
	minCount := int8(100)

	for i := gameMaJiang.MinHuaSe; i <= gameMaJiang.MaxHuaSe; i++ {
		if this.MJSeat.ShouPai[uint8(i)][0] < 1 || this.MJSeat.ShouPai[uint8(i)][0] < int8(count) {
			continue
		}
		if this.MJSeat.ShouPai[uint8(i)][0] < minCount {
			_minHuSe, minCount = i, this.MJSeat.ShouPai[uint8(i)][0]
		}
	}

	if _minHuSe >= gameMaJiang.MinHuaSe && _minHuSe <= gameMaJiang.MaxHuaSe {
		for j := gameMaJiang.MinDianShu_1; j <= gameMaJiang.MaxDianShu_9; j++ {
			for k := int8(0); k < this.MJSeat.ShouPai[_minHuSe][j]; k++ {
				groupArr = append(groupArr, (_minHuSe*0x10)|j)
				if len(groupArr) >= count {
					return groupArr
				}
			}
		}
	}

	return groupArr
}

func (this *XZDDSeat) GetAutoDingQue() int8 {

	minHuse := int8(0)
	minCount := int8(100)

	for i := gameMaJiang.MinHuaSe; i < gameMaJiang.MaxHuaSe; i++ {
		if this.MJSeat.ShouPai[uint8(i)][0] < minCount {
			minHuse, minCount = i, this.MJSeat.ShouPai[uint8(i)][0]
		}
	}

	return minHuse
}

func (this *XZDDSeat) GetXiQian() bool {
	if this.MJSeat.ShouPai[gameMaJiang.Zi>>4][gameMaJiang.Zhong&0x0F] == 4 {
		return true
	}
	return false
}

func (this *XZDDSeat) GetDingQuePaiCount() int8 {
	if this.DingQue < 0 {
		return 0
	}
	return this.MJSeat.ShouPai[this.DingQue][0]
}
