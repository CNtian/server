package ZhaJinHua

import (
	"fmt"
	"qpGame/localConfig"
	"qpGame/qpTable"
	"strconv"
)

type ZhaJinHuaRule struct {
	RuleJson        string  `json:"-"`         //
	MaxRoundCount   int32   `json:"maxCount"`  // 总局数 6\12
	MaxPlayer       int32   `json:"maxPlayer"` // 最大人数
	Multiple        string  `json:"multiple"`  // 倍数
	MultipleFloat64 float64 `json:"-"`
	TimeOut         int32   `json:"timeOut"`

	XiaZhuOpt int32 `json:"diFen"` // 下注 底分 // 0:1/2/5  1:2/5/10 2:5/10/20 3:10/20/50 20/50/100

	DaXiao          int32 `json:"daXiao"`          // 比大小的方式 0:比点数,1:全比(比花色)
	MenPaiRound     int32 `json:"menPai"`          // 闷牌轮数 0 2,3,5
	FengDingKaiPai  int32 `json:"fengDingKaiPai"`  // 5,10,15,20
	FirstRoundReady int32 `json:"firstRoundReady"` // 0(所有人),2 ,3 ,4

	BaoZiJiangLi   bool `json:"baoZiJiangLi"`   // 豹子额外奖励10倍底分
	ShuangBeiBiPai bool `json:"shuangBeiBiPai"` // 双倍比牌
	JieSanTongBi   bool `json:"jieSanTongBi"`   // 解散通比(中途解散是否要比牌)
	Second_32A     bool `json:"_32A"`           // AKQ>32A>KQJ

	// 服务器自定义(与客户端一致)
	DiZhu       [5][3]float64
	Consumables int32 `json:"consumables"` // 消耗(房卡|钻石)
}

func (this *ZhaJinHuaRule) CheckField() error {
	switch this.MaxRoundCount {
	case 6:
		this.Consumables = 4 // 2
	case 8:
		this.Consumables = 4 // 2
	case 12:
		this.Consumables = 8 // 2
	default:
		return fmt.Errorf("maxRound := %d", this.MaxRoundCount)
	}

	if this.MaxPlayer < 2 || this.MaxPlayer > 8 {
		return fmt.Errorf("maxPlayer := %d", this.MaxPlayer)
	}

	switch this.TimeOut {
	case 10, 15, 20:
	default:
		return fmt.Errorf("TimeOut := %d", this.TimeOut)
	}

	switch this.Multiple {
	case "0.1":
	case "0.2":
	//case "0.3":
	case "0.5":
	case "1":
	case "2":
	case "3":
	case "5":
	case "10":
	default:
		return fmt.Errorf("multiple := %s", this.Multiple)
	}
	this.MultipleFloat64, _ = strconv.ParseFloat(this.Multiple, 64)

	if this.XiaZhuOpt < 0 || this.XiaZhuOpt > 3 {
		return fmt.Errorf("xiazhu := %d", this.XiaZhuOpt)
	}

	switch this.XiaZhuOpt {
	case 0, 1, 2, 3:
	default:
		return fmt.Errorf("xiazhu := %d", this.XiaZhuOpt)
	}

	this.DiZhu[0][0] = 1
	this.DiZhu[0][1] = 2
	this.DiZhu[0][2] = 5

	this.DiZhu[1][0] = 2
	this.DiZhu[1][1] = 5
	this.DiZhu[1][2] = 10

	this.DiZhu[2][0] = 5
	this.DiZhu[2][1] = 10
	this.DiZhu[2][2] = 20

	this.DiZhu[3][0] = 10
	this.DiZhu[3][1] = 20
	this.DiZhu[3][2] = 50

	this.DiZhu[4][0] = 20
	this.DiZhu[4][1] = 50
	this.DiZhu[4][2] = 100

	if this.DaXiao != 0 && this.DaXiao != 1 {
		return fmt.Errorf("daxiao := %d", this.DaXiao)
	}

	switch this.MenPaiRound {
	case 0, 1, 2, 3:
	default:
		return fmt.Errorf("menPai := %d", this.MenPaiRound)
	}

	switch this.FengDingKaiPai {
	case 5, 10, 15, 20:
	default:
		return fmt.Errorf("fengDingKaiPai := %d", this.FengDingKaiPai)
	}

	switch this.FirstRoundReady {
	case 0, 2, 3, 4, 5, 6, 7:
	default:
		return fmt.Errorf("firstRoundReady := %d", this.FirstRoundReady)
	}

	return nil
}

func (this *ZhaJinHuaRule) GetMaxPlayerCount() int32 {
	return this.MaxPlayer
}

func newZhaJinHuaSeat(playerID qpTable.PlayerID, seatNumber qpTable.SeatNumber) (qpTable.QPSeat, error) {
	baseSeat := qpTable.NewQPSeat(playerID, seatNumber)

	gameSeat := &ZhaJinHuaSeat{seatData: baseSeat}
	gameSeat.CleanRoundData()

	return gameSeat, nil
}

func NewZhaJinHuaTable(tableNum int32, gameRuleCfg, tableCfg string) (*ZhaJinHuaTable, int32, string) {
	baseTable, rspCode, err := qpTable.NewQPTable(tableNum, tableCfg, newZhaJinHuaSeat)
	if err != nil {
		return nil, rspCode, err.Error()
	}

	t := ZhaJinHuaTable{}
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
	t.table.IsUnReady = false

	t.PaiMgr = NewPokerPaiBaseMgr(localConfig.GetConfig().IsTestPai)
	t.bankerSeatNumber = qpTable.INVALID_SEAT_NUMBER
	t.logic.rule = &t.gameRule
	t.PaiMgr.XiPai(t.table.GetCurSeatCount(), 3)

	t.CleanRoundData()
	return &t, rspCode, ""
}
