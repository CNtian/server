package NiuNiu_wzTongBi

import (
	"fmt"
	"qpGame/localConfig"
	"qpGame/qpTable"
	"strconv"
)

var normalBeiShuMap map[int32]float64
var superBeiShuMap map[int32]float64

func init() {
	normalBeiShuMap = map[int32]float64{
		TongHuaShun: 10,
		WuXiaoNiu:   9,
		ZhaDan:      8,
		WuHuaNiu:    7,
		TongHuaNiu:  6,
		ShunZiNiu:   5,
		HuLuNiu:     4,
		NiuNiu:      3,
		Niu_9:       2,
		Niu_8:       2,
		Niu_7:       2,
		Niu_6:       1,
		Niu_5:       1,
		Niu_4:       1,
		Niu_3:       1,
		Niu_2:       1,
		Niu_1:       1,
		Niu_0:       1}

	superBeiShuMap = map[int32]float64{
		TongHuaShun: 20,
		WuXiaoNiu:   19,
		ZhaDan:      18,
		WuHuaNiu:    17,
		TongHuaNiu:  16,
		ShunZiNiu:   15,
		HuLuNiu:     14,
		NiuNiu:      10,
		Niu_9:       9,
		Niu_8:       8,
		Niu_7:       7,
		Niu_6:       6,
		Niu_5:       5,
		Niu_4:       4,
		Niu_3:       3,
		Niu_2:       2,
		Niu_1:       1,
		Niu_0:       1}
}

type NiuNiuRule struct {
	RuleJson        string  `json:"-"`         //
	MaxRoundCount   int32   `json:"maxCount"`  // 总局数 10\20
	MaxPlayer       int32   `json:"maxPlayer"` // 最大人数 2\3人
	Multiple        string  `json:"multiple"`  // 倍数
	MultipleFloat64 float64 `json:"-"`

	XiaZhuOpt int32 `json:"xiaZhu"` // 下注 底分 // 0:1/2/3/4  1:5/10/15/20 2:10/20/50/100
	XiaZhuArr [3][4]int32

	//IsNormal bool `json:"isNormal"`
	IsSuper bool `json:"isSuper"`

	FirstRoundReady int32 `json:"firstRoundReady"` // 0(所有人),2 ,3 ,4

	// 服务器自定义(与客户端一致)
	Consumables int32 `json:"consumables"` // 消耗(房卡|钻石)
}

func (this *NiuNiuRule) CheckField() error {
	switch this.MaxRoundCount {
	case 1:
		this.Consumables = 1 // 1
	//case 20:
	//	this.Consumables = 2 // 2
	default:
		return fmt.Errorf("maxRound := %d", this.MaxRoundCount)
	}
	this.MaxPlayer = 8
	//if this.MaxPlayer < 2 || this.MaxPlayer > 8 {
	//	return fmt.Errorf("maxPlayer := %d", this.MaxPlayer)
	//}

	switch this.Multiple {
	case "0.1", "0.2", "0.3", "0.5":
	case "1", "2", "3", "5", "10", "20", "30":
	default:
		return fmt.Errorf("multiple := %s", this.Multiple)
	}
	this.MultipleFloat64, _ = strconv.ParseFloat(this.Multiple, 64)

	switch this.XiaZhuOpt {
	case 0, 1, 2:
	default:
		return fmt.Errorf("XiaZhuOpt := %d", this.XiaZhuOpt)
	}

	this.XiaZhuArr[0][0] = 1
	this.XiaZhuArr[0][1] = 2
	this.XiaZhuArr[0][2] = 3
	this.XiaZhuArr[0][3] = 4

	this.XiaZhuArr[1][0] = 5
	this.XiaZhuArr[1][1] = 10
	this.XiaZhuArr[1][2] = 15
	this.XiaZhuArr[1][3] = 20

	this.XiaZhuArr[2][0] = 10
	this.XiaZhuArr[2][1] = 20
	this.XiaZhuArr[2][2] = 30
	this.XiaZhuArr[2][3] = 40

	switch this.FirstRoundReady {
	case 0, 2, 3, 4, 5, 6, 7:
	default:
		return fmt.Errorf("firstRoundReady := %d", this.FirstRoundReady)
	}

	return nil
}

func (this *NiuNiuRule) GetMaxPlayerCount() int32 {
	return this.MaxPlayer
}

func newNiuNiuSeat(playerID qpTable.PlayerID, seatNumber qpTable.SeatNumber) (qpTable.QPSeat, error) {
	baseSeat := qpTable.NewQPSeat(playerID, seatNumber)

	pdkSeat := &NiuNiuSeat{seatData: baseSeat}
	pdkSeat.CleanRoundData()

	return pdkSeat, nil
}

func NewNiuNiuWuZhuangTongBiTable(tableNum int32, gameRuleCfg, tableCfg string) (*NiuNiuTable, int32, string) {
	baseTable, rspCode, err := qpTable.NewQPTable(tableNum, tableCfg, newNiuNiuSeat)
	if err != nil {
		return nil, rspCode, err.Error()
	}

	t := NiuNiuTable{}
	rspCode, err = t.ParseTableOptConfig(gameRuleCfg)
	if rspCode != 0 {
		return nil, rspCode, err.Error()
	}

	baseTable.GameOverFunc = func() {
		t.handleXiaoJieSuan()
		t.handleDaJieSuan()
	}
	baseTable.Consumables = t.gameRule.Consumables
	baseTable.TableRule.Table = &t
	t.table = baseTable
	t.table.SetMaxPlayers(t.gameRule.MaxPlayer)

	t.PaiMgr = NewPokerPaiBaseMgr(localConfig.GetConfig().IsTestPai)

	//t.logic.SetRule(&t.gameRule)

	t.CleanRoundData()
	return &t, rspCode, ""
}
