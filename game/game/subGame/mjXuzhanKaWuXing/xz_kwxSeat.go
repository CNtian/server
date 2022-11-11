package mj_XueZhan_KWXTable

import (
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
)

const SS_Suspend qpTable.SeatStatus = 64 // 暂停

type huData struct {
	pai   int8
	score float64
}

type KWXSeat struct {
	MJSeat *gameMaJiang.MJSeat

	//totalPaiXinScore float64
	//totalGangScore   float64

	//LiangDaoScore float64 // 亮倒 输赢分(流局)
	//PaiXingScore  float64 // 胡时，牌分

	GangScore   float64       // 杠
	LiangDaoMap map[int8]int8 // 亮倒牌 key:牌 value:数量
	KouMap      map[int8]int8 // 扣牌(必定是3张) key:牌 value:数量
	TingPaiMap  map[int8]int8 // 听牌

	HuPaiArr    []int8 // 胡的牌
	huCount     int
	firstHuTime int64
}

// 清理座位一轮数据
func (this *KWXSeat) CleanRoundData() {
	this.GangScore = 0
	this.LiangDaoMap = nil
	this.KouMap = nil
	this.TingPaiMap = nil
	this.HuPaiArr = []int8{}
	this.huCount = 0
	this.firstHuTime = 0
	this.MJSeat.SeatData.DelState(SS_Suspend)
	this.MJSeat.CleanRoundData()
}

func (this *KWXSeat) GetSeatData() *qpTable.SeatData {
	return this.MJSeat.GetSeatData()
}

func (this *KWXSeat) GetXSeatData(value int) interface{} {
	if value == 0 {
		return this.MJSeat
	}
	return nil
}

func (this *KWXSeat) GetLiangDaoPai() []int8 {
	tempArr := make([]int8, 0, 14)
	for k, v := range this.LiangDaoMap {
		for i := int8(0); i < v; i++ {
			tempArr = append(tempArr, k)
		}
	}
	return tempArr
}

func (this *KWXSeat) GetTingPai() []int8 {
	tempArr := make([]int8, 0, 14)
	for k, v := range this.TingPaiMap {
		for i := int8(0); i < v; i++ {
			tempArr = append(tempArr, k)
		}
	}
	return tempArr
}

func (this *KWXSeat) GetKouPai() []int8 {
	tempArr := make([]int8, 0, 14)
	for k, v := range this.KouMap {
		for i := int8(0); i < v; i++ {
			tempArr = append(tempArr, k)
		}
	}
	return tempArr
}
