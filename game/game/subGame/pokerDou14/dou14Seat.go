package pokerDouFourteen

import (
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
	"sort"
)

type SortPai []int8

func (s SortPai) Len() int      { return len(s) }
func (s SortPai) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortPai) Less(i, j int) bool {
	return s[i] < s[j]
}

const SS_Bao = qpTable.SS_CustomDefineBase //自定义状态起始值

type PaiMap map[int8]struct{} // key:牌值

type PokerDou14Seat struct {
	seatData *qpTable.SeatData
	paiLogic dou14Logic

	hupaiCount, dianPaoCount, ziMoCount int
	_gangInfo                           gangInfo
	// 小局清理
	shouPaiMap       map[int8]PaiMap // key:点数 value:
	curOperationItem D14Operation
	isXiaoJia        bool // 小家

	chiPai  []SortPai // 吃
	pengPai []SortPai // 碰
	gangPai []SortPai // 杠
	touPai  []SortPai // 偷
	playPai []int8    // 出
	canGang []int8    // 可杠的牌

	lianGang     int64
	lastIsTouPai bool
	chuPaiCount  int
	moPaiCount   int

	reserveShouPai []int8 // 测试手牌
	reserveMoPai   int8   // 测试摸牌

	baoJiaoed bool // 是否已经操作过报叫
	huPaiXin  []gameMaJiang.HuPaiXing
	gameScore int
	diScore   int
}

// 清理座位一轮数据
func (this *PokerDou14Seat) CleanRoundData() {
	this.shouPaiMap = make(map[int8]PaiMap)
	this.curOperationItem = 0
	this.isXiaoJia = false

	this.chiPai = make([]SortPai, 0, 7)
	this.pengPai = make([]SortPai, 0, 3)
	this.gangPai = make([]SortPai, 0, 3)
	this.touPai = make([]SortPai, 0, 3)
	this.playPai = make(SortPai, 0, 10)
	this.canGang = nil
	this.lianGang = 0
	this.lastIsTouPai = false
	this.chuPaiCount = 0
	this.moPaiCount = 0
	this.baoJiaoed = false
	//this.reserveShouPai = nil
	this.reserveMoPai = InvalidPai
	this.huPaiXin = nil
	this.gameScore, this.diScore = 0, 0
	this.seatData.DelState(SS_Bao)
	this.seatData.CleanRoundData()
}

func (this *PokerDou14Seat) GetSeatData() *qpTable.SeatData {
	return this.seatData
}

func (this *PokerDou14Seat) GetXSeatData(int) interface{} {
	return this
}

func (this *PokerDou14Seat) SetOperationItem(value D14Operation) {

	this.seatData.MakeOperationID()
	this.curOperationItem = value
}

func (this *PokerDou14Seat) GetShouPaiBak() map[int8]PaiMap {
	shouPaiBak := make(map[int8]PaiMap)

	for _, v := range this.shouPaiMap {
		for k, _ := range v {
			v, ok := shouPaiBak[k&0x0F]
			if ok == false {
				v = make(PaiMap)
			}
			v[k] = struct{}{}
			shouPaiBak[k&0x0F] = v
		}
	}
	return shouPaiBak
}

// 添加手牌
func (this *PokerDou14Seat) PushShouPai(pai int8) {
	this.reserveMoPai = InvalidPai
	v, ok := this.shouPaiMap[pai&0x0F]
	if ok == false {
		v = make(PaiMap)
	}
	v[pai] = struct{}{}
	this.shouPaiMap[pai&0x0F] = v
}

// 删除手牌
func (this *PokerDou14Seat) DeleteShouPai(pai int8) bool {
	v, ok := this.shouPaiMap[pai&0x0F]
	if ok == false {
		return false
	}
	if _, ok = v[pai]; ok == false {
		return false
	}
	delete(v, pai)
	if len(v) < 1 {
		delete(this.shouPaiMap, pai&0x0F)
	} else {
		this.shouPaiMap[pai&0x0F] = v
	}
	return true
}

// 牌 是否存在
func (this *PokerDou14Seat) IsExist(pai int8) bool {

	if v, ok := this.shouPaiMap[pai&0x0F]; ok == true {
		if _, ok = v[pai]; ok == true {
			return true
		}
	}
	return false
}

// 手牌 数量
func (this *PokerDou14Seat) GetShouPaiCount() int {
	c := 0
	for _, v := range this.shouPaiMap {
		c += len(v)
	}
	return c
}

// 所有手牌
func (this *PokerDou14Seat) GetAllPai() []int8 {
	paiArr := make([]int8, 0, 16)
	for _, v := range this.shouPaiMap {
		for k, _ := range v {
			paiArr = append(paiArr, k)
		}
	}

	return paiArr
}

func (this *PokerDou14Seat) isExistLaiZi() bool {
	if v, _ := this.shouPaiMap[0]; len(v) > 0 {
		return true
	}
	return false
}

// 添加吃牌
func (this *PokerDou14Seat) PutChi(chiPai SortPai) {
	sort.Sort(chiPai)
	this.chiPai = append(this.chiPai, chiPai)
}

// 添加碰牌
func (this *PokerDou14Seat) PutPeng(pengPai SortPai) {
	sort.Sort(pengPai)
	this.pengPai = append(this.pengPai, pengPai)
}

// 添加杠牌
func (this *PokerDou14Seat) PutGang(gangPai SortPai) {
	sort.Sort(gangPai)

	this.gangPai = append(this.gangPai, gangPai)
}

// 添加偷牌
func (this *PokerDou14Seat) PutTou(touPai SortPai) {
	sort.Sort(touPai)

	this.touPai = append(this.touPai, touPai)
}

// 是否有偷牌
func (this *PokerDou14Seat) hasTouMo(fanpai int8) bool {
	if fanpai != InvalidPai {
		if fanpai == DaWang {
			return true
		}
		if fanpai == XiaoWang {
			return true
		}
		if fanpai&0x0F == 7 {
			return true
		}

		return false
	}

	// 手上 是否有 可偷的
	if v, ok := this.shouPaiMap[DaWang&0x0F]; ok == true && len(v) > 0 {
		return true
	}
	if v, ok := this.shouPaiMap[XiaoWang&0x0F]; ok == true && len(v) > 0 {
		return true
	}
	if v, ok := this.shouPaiMap[LaiZi&0x0F]; ok == true && len(v) > 0 {
		return true
	}
	if v, ok := this.shouPaiMap[7]; ok == true && len(v) > 0 {
		return true
	}

	for _, v := range this.shouPaiMap {
		if len(v) >= 3 {
			return true
		}
	}

	return false
}

func (this *PokerDou14Seat) hasGang(fanPai, chuPai int8, fanPaiSeatNo qpTable.SeatNumber) {
	this.canGang = nil
	isLaizi := false //this.isExistLaiZi()

	if fanPai != InvalidPai {
		if fanPaiSeatNo == this.seatData.Number {
			for _, v := range this.pengPai {
				if v[0]&0x0F == fanPai&0x0F {
					this.canGang = []int8{fanPai}
					return
				}

			}
			for _, v := range this.touPai {
				if len(v) == 3 {
					if v[0]&0x0F == fanPai&0x0F {
						this.canGang = []int8{fanPai}
						return
					}
				}
			}
		}

		paiMap_, ok := this.shouPaiMap[fanPai&0x0F]
		if ok == false {
			return
		}

		if isLaizi == true {
			if len(paiMap_) >= 2 {
				this.canGang = []int8{fanPai}
				return
			}
		} else {
			if len(paiMap_) >= 3 {
				this.canGang = []int8{fanPai}
				return
			}
		}
		return
	}

	if chuPai != InvalidPai {
		paiMap_, ok := this.shouPaiMap[chuPai&0x0F]
		if ok == false {
			return
		}

		if isLaizi == true {
			if len(paiMap_) >= 2 {
				this.canGang = []int8{chuPai}
				return
			}
		} else {
			if len(paiMap_) >= 3 {
				this.canGang = []int8{chuPai}
				return
			}
		}
		return
	}

	// 手上

	gangArr := make([]int8, 0, 3)
	// 手上有赖子 + 有碰牌
	if isLaizi {
		for _, v := range this.pengPai {
			gangArr = append(gangArr, v[0])
		}
		for _, v := range this.touPai {
			if len(v) == 3 {
				gangArr = append(gangArr, v[0])
			}
		}
	}

	for k, v := range this.shouPaiMap {
		// 碰后, 手上还有一张
		for i, _ := range this.pengPai {
			if this.pengPai[i][0]&0x0F == k {
				gangArr = append(gangArr, k)
			}
		}
		// 偷后, 手上还有一张
		for i, _ := range this.touPai {
			if len(this.touPai[i]) == 3 &&
				this.touPai[i][0]&0x0F == k {
				gangArr = append(gangArr, k)
			}
		}

		// 碰+赖子 ->杠  ->超级杠
		for i, _ := range this.gangPai {
			if this.gangPai[i][0]&0x0F == k {
				gangArr = append(gangArr, k)
			}
		}

		if isLaizi == true {
			if len(v) >= 3 {
				gangArr = append(gangArr, k)
			}
		} else {
			if len(v) >= 4 {
				gangArr = append(gangArr, k)
			}
		}
	}

	this.canGang = gangArr
	return
}

func (this *PokerDou14Seat) hasChi(playPai int8) bool {
	if playPai == InvalidPai {
		return false
	}
	if playPai == DaWang || playPai == XiaoWang {
		return false
	}

	if this.GetShouPaiCount() < 2 {
		return false
	}

	if this.isExistLaiZi() == true {
		return true
	}

	a := 14 - playPai&0x0F
	tempArr := this.shouPaiMap[a]
	if len(tempArr) > 0 {
		return true
	}
	return false
}

func (this *PokerDou14Seat) hasPeng(playPai int8) bool {
	if playPai == InvalidPai {
		return false
	}

	paiMap_, ok := this.shouPaiMap[playPai&0x0F]
	if ok == false {
		return false
	}

	if this.isExistLaiZi() == true {
		if len(paiMap_) >= 1 {
			return true
		}
	} else {
		if len(paiMap_) >= 2 {
			return true
		}
	}

	return false
}

// :是否使用赖子,是否成功
func (this *PokerDou14Seat) checkChi(chiPai int8, chu_fan_Pai int8) (bool, bool) {

	if this.IsExist(chiPai) == false {
		return false, false
	}

	if chiPai == LaiZi {
		return true, true
	}

	if ((chiPai & 0x0F) + (chu_fan_Pai & 0x0F)) == 14 {
		return false, true
	}

	return false, false
}

// :是否使用赖子,是否成功
func (this *PokerDou14Seat) checkPeng(pengPai SortPai, chu_fan_Pai int8) (bool, bool) {
	if len(pengPai) != 2 {
		return false, false
	}

	for _, v := range pengPai {
		if this.IsExist(v) == false {
			return false, false
		}
	}

	sort.Sort(pengPai)

	_1Dian := pengPai[0] & 0x0F
	_2Dian := pengPai[1] & 0x0F
	_3Dian := chu_fan_Pai & 0x0F

	if _1Dian == _3Dian && _2Dian == _3Dian {
		return false, true
	}
	if _1Dian == _3Dian && pengPai[1] == LaiZi {
		return true, true
	}

	return false, false
}

// :是否使用碰,杠类型[1:碰+翻牌 2:碰+手牌  3:手牌+翻牌 4:手牌 5:手牌+出牌],是否成功
type gangInfo struct {
	PengIndex  int
	TouIndex   int
	GangIndex  int
	Category   int
	ChuPai     int8
	FanPai     int8
	useShouPai int

	gangPaiArr    []int8
	isQiangGangHu bool
	ptgBak_       []int8 // 备份
}

func (this *PokerDou14Seat) checkGang(gangPai SortPai, info *gangInfo) bool {

	info.PengIndex, info.TouIndex = -1, -1

	if info.FanPai != InvalidPai {
		return this.checkGangFanPai(gangPai, info)
	}
	if info.ChuPai != InvalidPai {
		return this.checkGangChuPai(gangPai, info)
	}

	return this.checkGangShouPai(gangPai, info)
}

func (this *PokerDou14Seat) checkGangFanPai(gangPai SortPai, info *gangInfo) bool {

	// 操作区域的牌   碰/偷+翻 -> 杠

	info.gangPaiArr = []int8{}

	for i, v := range this.pengPai {
		if v[0]&0x0F == info.FanPai&0x0F {
			info.PengIndex, info.Category = i, 1
			info.gangPaiArr = this.pengPai[i]
			info.gangPaiArr = append(info.gangPaiArr, info.FanPai)
			return true
		}
	}
	for i, v := range this.touPai {
		if len(v) == 3 {
			if v[0]&0x0F == info.FanPai&0x0F {
				info.TouIndex, info.Category = i, 1
				info.gangPaiArr = this.touPai[i]
				info.gangPaiArr = append(info.gangPaiArr, info.FanPai)
				return true
			}
		}
	}

	inShouPaiArr := make(SortPai, 0, 4)
	for _, v := range gangPai {
		if this.IsExist(v) == true {
			inShouPaiArr = append(inShouPaiArr, v)
		}
	}
	if len(inShouPaiArr) != 3 {
		return false
	}
	info.useShouPai = len(inShouPaiArr)

	sort.Sort(inShouPaiArr)

	if inShouPaiArr[0]&0x0F != info.FanPai&0x0F ||
		inShouPaiArr[1]&0x0F != info.FanPai&0x0F {
		return false
	}
	if inShouPaiArr[2]&0x0F == info.FanPai&0x0F ||
		inShouPaiArr[2] == LaiZi {
		info.Category = 3 // 手牌 + 翻牌

		inShouPaiArr = append(inShouPaiArr, info.FanPai)
		info.gangPaiArr = inShouPaiArr
		return true
	}

	return false
}

func (this *PokerDou14Seat) checkGangChuPai(gangPai SortPai, info *gangInfo) bool {

	info.gangPaiArr = []int8{}

	inShouPaiArr := make(SortPai, 0, 4)
	for _, v := range gangPai {
		if this.IsExist(v) == true {
			inShouPaiArr = append(inShouPaiArr, v)
		}
	}
	if len(inShouPaiArr) != 3 {
		return false
	}

	info.useShouPai = len(inShouPaiArr)
	sort.Sort(inShouPaiArr)

	// 出的牌, 只能杠手牌

	if len(inShouPaiArr) != 3 {
		return false
	}
	if inShouPaiArr[0]&0x0F != info.ChuPai&0x0F ||
		inShouPaiArr[1]&0x0F != info.ChuPai&0x0F {
		return false
	}
	if inShouPaiArr[2]&0x0F == info.ChuPai&0x0F ||
		inShouPaiArr[2] == LaiZi {
		info.Category = 5 // 手牌 + 出牌
		inShouPaiArr = append(inShouPaiArr, info.ChuPai)
		info.gangPaiArr = inShouPaiArr
		return true
	}

	return false
}

func (this *PokerDou14Seat) checkGangShouPai(gangPai SortPai, info *gangInfo) bool {

	info.gangPaiArr = []int8{}

	inShouPaiArr := make(SortPai, 0, 4)
	for _, v := range gangPai {
		if this.IsExist(v) == true {
			if v == LaiZi {
				return false
			}
			inShouPaiArr = append(inShouPaiArr, v)
		}
	}

	info.useShouPai = len(inShouPaiArr)

	var flagPai int8
	if len(inShouPaiArr) == 1 {
		if inShouPaiArr[0] == LaiZi {
			sort.Sort(gangPai)
			flagPai = gangPai[0]
		} else {
			flagPai = inShouPaiArr[0]
		}
	}

	if len(inShouPaiArr) == 1 {
		for i, v := range this.pengPai {
			if v[0]&0x0F == flagPai&0x0F {
				info.PengIndex, info.Category = i, 2 // 碰/偷 + 手牌

				info.gangPaiArr = this.pengPai[i]
				info.gangPaiArr = append(info.gangPaiArr, inShouPaiArr[0])
				return true
			}
		}
		for i, v := range this.touPai {
			if len(v) == 3 {
				if v[0]&0x0F == flagPai&0x0F {
					info.TouIndex, info.Category = i, 2 // 碰/偷 + 手牌
					info.gangPaiArr = this.touPai[i]
					info.gangPaiArr = append(info.gangPaiArr, inShouPaiArr[0])
					return true
				}
			}
		}

		for i, v := range this.gangPai {
			if len(v) == 4 {
				if v[0]&0x0F == flagPai&0x0F {
					info.GangIndex, info.Category = i, 2 // 杠 + 手牌
					info.gangPaiArr = this.gangPai[i]
					info.gangPaiArr = append(info.gangPaiArr, inShouPaiArr[0])
					return true
				}
			}
		}
	}

	if len(inShouPaiArr) == 4 {
		flagPai = inShouPaiArr[0]

		if inShouPaiArr[1]&0x0F != flagPai&0x0F ||
			inShouPaiArr[2]&0x0F != flagPai&0x0F {
			return false
		}
		if inShouPaiArr[3]&0x0F == flagPai&0x0F ||
			inShouPaiArr[3] == LaiZi {
			info.Category = 4 // 手牌
			info.gangPaiArr = inShouPaiArr
			return true
		}
	}
	return false
}

func (this *PokerDou14Seat) checkTouPai(paiArr SortPai, fan_pai int8) (bool, bool) {
	if fan_pai != InvalidPai {
		return false, true
	}

	sort.Sort(paiArr)

	for _, v := range paiArr {
		if this.IsExist(v) == false {
			return false, false
		}
	}

	if len(paiArr) == 1 {
		_d := paiArr[0] & 0x0F
		// 7 大小王
		if _d == 7 || _d == 0x0E || _d == 0x0F {
			return false, true
		}
		// 赖子
		if paiArr[0] == LaiZi {
			return true, true
		}
		return false, false
	}
	if len(paiArr) == 3 {
		flagPai := paiArr[0]

		if paiArr[1]&0x0F != flagPai&0x0F {
			return false, false
		}
		if paiArr[2]&0x0F == flagPai&0x0F {
			return false, true
		}
		if paiArr[2] == LaiZi {
			return true, true
		}
		return false, false
	}

	return false, false
}

func (this *PokerDou14Seat) getShuangJinShuangChu(dW, xW *bool) int {
	pvMap := make(map[int8]int)

	for _, v := range this.shouPaiMap {
		for k, _ := range v {
			pvMap[k&0x0F] += 1
			if k == DaWang {
				*dW = true
			} else if k == XiaoWang {
				*xW = true
			}
		}
	}
	for _, v := range this.touPai {
		for _, vv := range v {
			pvMap[vv&0x0F] += 1
			if vv == DaWang {
				*dW = true
			} else if vv == XiaoWang {
				*xW = true
			}
		}
	}

	for _, v := range this.chiPai {
		for _, vv := range v {
			pvMap[vv&0x0F] += 1
		}
	}

	for _, v := range this.pengPai {
		pvMap[v[0]&0x0F] += 3
	}

	for _, v := range this.gangPai {
		pvMap[v[0]&0x0F] += 4
	}

	i := 0
	for _, v := range pvMap {
		if v >= 4 {
			i++
		}
	}

	return i
}

/*
func (A *PokerDou14Seat) FindGreaterPai(playSeat *PokerDou14Seat, rule *PDKRule) bool {
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
