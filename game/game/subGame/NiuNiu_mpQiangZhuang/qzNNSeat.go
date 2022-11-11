package NiuNiu_mpQiangZhuang

import "qpGame/qpTable"

type NiuNiuMPQZSeat struct {
	seatData    *qpTable.SeatData
	maxPaiXing  int32
	zhuangCount int32
	tuiZhuCount int32
	pro         int

	// 小局清理
	qiangZhuang int32
	xiaZhu      int32
	shouPai     []int8
	paiXing     int32
	maxPai      int8 // 最大牌
	isLiang     bool // 是否亮牌
	roundScore  float64
}

func (this *NiuNiuMPQZSeat) CleanRoundData() {
	this.qiangZhuang = -1
	this.xiaZhu = 0
	this.shouPai = nil
	this.maxPai = 0
	this.isLiang = false
	this.roundScore = 0
	this.paiXing = NULL
	this.seatData.CleanRoundData()
}

func (this *NiuNiuMPQZSeat) GetSeatData() *qpTable.SeatData {
	return this.seatData
}

func (this *NiuNiuMPQZSeat) GetXSeatData(int) interface{} {
	return this
}

func (this *NiuNiuMPQZSeat) SetShouPai(paiArr []int8) {
	this.shouPai = paiArr
}
