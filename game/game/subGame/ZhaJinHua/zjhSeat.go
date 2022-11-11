package ZhaJinHua

import "qpGame/qpTable"

type ZhaJinHuaSeat struct {
	seatData    *qpTable.SeatData
	MaxPaiXing  int32
	MaxGetScore float64
	WinCount    int32
	pro         int

	// 小局清理
	//curLevelXiaZhu int32
	//curIndexXiaZhu int32
	xiaZhuTime  int32 // 下注次数
	XiaZhuScore float64
	shouPai     []int8
	zjhPaiXing  *zhaJinHuaPaiXing
	isKanPai    bool // 是否看牌
	isBiPaiLose bool // 比牌是否输了
	isQiPai     bool // 是否弃牌
	isGenDaoDi  bool // 跟到底
}

func (this *ZhaJinHuaSeat) CleanRoundData() {
	//this.curLevelXiaZhu,this.curIndexXiaZhu = 0,0
	this.xiaZhuTime = 0
	this.XiaZhuScore = 0.0
	this.shouPai = nil
	this.zjhPaiXing = nil
	this.isKanPai = false
	this.isBiPaiLose = false
	this.isQiPai = false
	this.isGenDaoDi = false

	this.seatData.CleanRoundData()
}

func (this *ZhaJinHuaSeat) GetSeatData() *qpTable.SeatData {
	return this.seatData
}

func (this *ZhaJinHuaSeat) GetXSeatData(int) interface{} {
	return this
}

func (this *ZhaJinHuaSeat) SetShouPai(pai []int8) {
	this.shouPai = pai
}
