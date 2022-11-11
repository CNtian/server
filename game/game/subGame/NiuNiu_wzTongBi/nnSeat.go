package NiuNiu_wzTongBi

import "qpGame/qpTable"

type NiuNiuSeat struct {
	seatData   *qpTable.SeatData
	maxPaiXing int32
	winCount   int32

	// 小局清理
	xiaZhu  int32
	shouPai []int8
	paiXing int32
	maxPai  int8 // 最大牌
	isLiang bool // 是否亮牌
}

func (this *NiuNiuSeat) CleanRoundData() {
	this.xiaZhu = 0
	this.shouPai = nil
	this.maxPai = 0
	this.isLiang = false
	this.paiXing = NULL
	this.seatData.CleanRoundData()
}

func (this *NiuNiuSeat) GetSeatData() *qpTable.SeatData {
	return this.seatData
}

func (this *NiuNiuSeat) GetXSeatData(int) interface{} {
	return this
}

func (this *NiuNiuSeat) SetShouPai(paiArr []int8) {
	this.shouPai = paiArr
}
