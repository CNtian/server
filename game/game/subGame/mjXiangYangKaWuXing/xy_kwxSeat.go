package mjXYKWXTable

import (
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
)

type KWXSeat struct {
	MJSeat *gameMaJiang.MJSeat

	totalMaScore     float64
	totalPaiXinScore float64
	totalGangScore   float64
	totalPiaoScore   float64

	MaScore       float64 // 码分 输赢分(抓码)
	LiangDaoScore float64 // 亮倒 输赢分(流局)
	OverPiaoScore float64 // 游戏结束时 漂输赢分
	PaiXingScore  float64 // 胡时，牌分

	GangScore   float64       // 杠
	PiaoScore   int64         // 漂分
	LiangDaoMap map[int8]int8 // 亮倒牌 key:牌 value:数量
	KouMap      map[int8]int8 // 扣牌(必定是3张) key:牌 value:数量
	TingPaiMap  map[int8]int8 // 听牌

	PaoScore float64
	QiaScore float64
	MoScore  float64
	BaScore  float64
	qiaCount int64
	baCount  int64

	delayRec []func()
}

// 清理座位一轮数据
func (this *KWXSeat) CleanRoundData() {
	this.MaScore = 0
	this.LiangDaoScore = 0
	this.OverPiaoScore = 0
	this.PaiXingScore = 0
	this.GangScore = 0
	this.PiaoScore = -1
	this.LiangDaoMap = nil
	this.KouMap = nil
	this.TingPaiMap = nil
	this.qiaCount = 0
	this.baCount = 0
	this.PaoScore = 0
	this.QiaScore = 0
	this.MoScore = 0
	this.BaScore = 0
	this.delayRec = make([]func(), 0, 1)
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
