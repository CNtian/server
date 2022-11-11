package pokerDouDiZhu

import (
	"qpGame/qpTable"
)

const (
	ZhaDan = iota + 1
)

type DouDiZhuSeat struct {
	seatData       *qpTable.SeatData
	paiLogic       PokerDouDizhuLogic
	bombCount      int32
	winCount       int32
	loseCount      int32
	chunTianCount  int32
	TotalBombScore float64 // 炸弹分
	TotalPaiScore  float64 // 牌分

	autoPlayPaiArr []int8 // 自动出牌

	// 小局清理
	nextShouPai      []int8        // 下次手牌
	isChunTian       bool          // 是否是春天
	isFanChun        bool          // 是否反春
	bombScore        float64       // 炸弹分
	playPaiCount     int32         // 出牌次数
	shouPaiCount     int32         // 手牌数量
	shouPai          map[int8]int  // 手牌 key:牌  value:数量
	PlayPai          []int8        // 出牌
	CurOperationItem OperationItem // 操作项
	recChuPai        []int8        // 记录出牌
	PaiScore         float64       // 牌的输赢
}

// 清理座位一轮数据
func (this *DouDiZhuSeat) CleanRoundData() {
	this.nextShouPai = nil
	this.isChunTian = false
	this.isFanChun = false
	this.bombScore = 0
	this.playPaiCount = 0
	this.shouPaiCount = 0
	this.shouPai = make(map[int8]int)
	this.PlayPai = []int8{}
	this.CurOperationItem = 0
	this.recChuPai = make([]int8, 0, 16)
	this.PaiScore = 0

	this.seatData.CleanRoundData()
}

func (this *DouDiZhuSeat) GetSeatData() *qpTable.SeatData {
	return this.seatData
}

func (this *DouDiZhuSeat) GetXSeatData(int) interface{} {
	return this
}

func (this *DouDiZhuSeat) SetOperationItem(value OperationItem) {

	this.seatData.MakeOperationID()
	this.CurOperationItem = value
}

// 添加手牌
func (this *DouDiZhuSeat) PushShouPai(pai int8) {
	count, ok := this.shouPai[pai]
	if ok == false {
		this.shouPai[pai] = 1
	} else {
		this.shouPai[pai] = count + 1
	}
	this.shouPaiCount += 1
}

// 删除手牌
func (this *DouDiZhuSeat) DeleteShouPai(pai int8) bool {
	count, ok := this.shouPai[pai]
	if ok == true && count > 0 {
		this.shouPai[pai] = count - 1
		this.shouPaiCount -= 1

		this.recChuPai = append(this.recChuPai, pai)
		return true
	}
	return false
}

// 牌 是否存在
func (this *DouDiZhuSeat) GetPaiCount(pai int8) int {
	if count, ok := this.shouPai[pai]; ok == true {
		return count
	}
	return 0
}

func (this *DouDiZhuSeat) GetAllPai() []int8 {
	paiArr := make([]int8, 0, 16)
	for k, v := range this.shouPai {
		for i := 0; i < v; i++ {
			paiArr = append(paiArr, k)
		}
	}
	return paiArr
}

/*
func (A *DouDiZhuSeat) FindGreaterPai(playSeat *DouDiZhuSeat, rule *PDKRule) bool {
	shouPaiDianShuArr := [128]int8{}
	for k, v := range A.shouPai {
		shouPaiDianShuArr[k&0x0F] += int8(v)
	}

	findBombFunc := func(dianShu int8) bool {
		if rule.Is3ABomb == true && shouPaiDianShuArr[pokerTable.ADianShu] == 3 {
			return true
		}
		for i := dianShu; i <= pokerTable.MaxDianShu; i++ {
			if shouPaiDianShuArr[i] > 3 {
				return true
			}
		}
		return false
	}

	switch playSeat.paiLogic.PaiXing {
	case PDK_PX_ZhaDan:
		fallthrough
	case PDK_PX_SiDaiEr:
		fallthrough
	case PDK_PX_SiDaiSan:
		if findBombFunc(playSeat.paiLogic.PaiXingStartDianShu+1) == true {
			return true
		}
	case PDK_PX_FeiJi:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := playSeat.paiLogic.PaiXingStartDianShu + 1; i < pokerTable.MaxDianShu; i++ {
			if shouPaiDianShuArr[i] < 3 {
				continue
			}

			var tempCC int32
			for j := i + 1; j < pokerTable.MaxDianShu; j++ {
				if shouPaiDianShuArr[j] < 3 {
					break
				}
				tempCC += 1
			}
			if tempCC >= int32(playSeat.paiLogic.SequenceCount) {
				return true
			}
		}
	case PDK_PX_LianDui:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := playSeat.paiLogic.PaiXingStartDianShu + 1; i < pokerTable.MaxDianShu; i++ {
			if shouPaiDianShuArr[i] < 2 {
				continue
			}

			var tempCC int
			for j := i + 1; j < pokerTable.MaxDianShu; j++ {
				if shouPaiDianShuArr[j] < 2 {
					break
				}
				tempCC += 1
			}
			if tempCC >= playSeat.paiLogic.SequenceCount {
				return true
			}
		}
	case PDK_PX_ShunZi:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := playSeat.paiLogic.PaiXingStartDianShu + 1; i < pokerTable.MaxDianShu; i++ {
			if shouPaiDianShuArr[i] < 1 {
				continue
			}

			var tempCC int
			for j := i + 1; j < pokerTable.MaxDianShu; j++ {
				if shouPaiDianShuArr[j] < 1 {
					break
				}
				tempCC += 1
			}
			if tempCC >= playSeat.paiLogic.SequenceCount {
				return true
			}
		}
	case PDK_PX_SanDai_Er:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := playSeat.paiLogic.PaiXingStartDianShu + 1; i < pokerTable.MaxDianShu; i++ {
			if shouPaiDianShuArr[i] < 3 {
				continue
			}
			return true
		}
	case PDK_PX_YiDui:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := playSeat.paiLogic.PaiXingStartDianShu + 1; i < pokerTable.MaxDianShu; i++ {
			if shouPaiDianShuArr[i] < 2 {
				continue
			}
			return true
		}
	case PDK_PX_DandZhang:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := playSeat.paiLogic.PaiXingStartDianShu + 1; i <= pokerTable.MaxDianShu; i++ {
			if shouPaiDianShuArr[i] < 1 {
				continue
			}
			return true
		}
	}
	return false
}
*/
