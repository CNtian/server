package pokerDouDiZhu

import (
	"fmt"
	pokerTable "qpGame/game/poker"
	"qpGame/localConfig"
	"qpGame/qpTable"
	"strconv"
)

type PokerDouDiZhuRule struct {
	RuleJson        string  `json:"-"`         //
	MaxRoundCount   int32   `json:"maxCount"`  // 总局数 6\12
	MaxPlayer       int32   `json:"maxPlayer"` // 最大人数 2\3人
	Multiple        string  `json:"multiple"`  // 倍数 0.5\1\2\3\4\5\10\20\30\50
	MultipleFloat64 float64 `json:"-"`

	DiZhuMode int `json:"diZhu"`    // 0:叫分模式  1:赢家
	LaiZiMode int `json:"laiZi"`    // 0:无癞 1:广告癞 2:随机癞 3:广告+随机癞
	FengDing  int `json:"FDFanShu"` // 0:不封顶  8: 8倍  16: 16倍  32: 32倍 64: 64倍

	WwLaiZiDiZhu bool `json:"twoWangGGBJ"`  // 两王+广告牌 必叫 地主
	WwDiZhu      bool `json:"shuangWangBJ"` // 两王 必叫 地主
	IsShowPaiShu bool `json:"showPaiShu"`   // 显示手牌数量

	// 服务器自定义(与客户端一致)
	Consumables int32 `json:"consumables"` // 消耗(房卡|钻石)
}

func (this *PokerDouDiZhuRule) CheckField() error {
	switch this.MaxRoundCount {
	case 6, 8:
		this.Consumables = 1 // 1
	case 16:
		this.Consumables = 2 // 2
	default:
		return fmt.Errorf("maxRound := %d", this.MaxRoundCount)
	}

	if this.MaxPlayer != 2 && this.MaxPlayer != 3 {
		return fmt.Errorf("maxPlayer := %d", this.MaxPlayer)
	}

	if this.DiZhuMode != 0 && this.DiZhuMode != 1 {
		return fmt.Errorf("diZhu := %d", this.DiZhuMode)
	}

	if this.LaiZiMode < 0 || this.LaiZiMode > 3 {
		return fmt.Errorf("laiZi := %d", this.LaiZiMode)
	}

	switch this.FengDing {
	case 0, 8, 16, 32, 64:
	default:
		return fmt.Errorf("fengding := %d", this.FengDing)
	}

	switch this.Multiple {
	case "0.5":
	case "1":
	case "2":
	case "3":
	case "4":
	case "5":
	case "10":
	case "20":
	case "30":
	case "50":
	default:
		return fmt.Errorf("multiple := %s", this.Multiple)
	}
	this.MultipleFloat64, _ = strconv.ParseFloat(this.Multiple, 64)

	return nil
}

func (this *PokerDouDiZhuRule) GetMaxPlayerCount() int32 {
	return this.MaxPlayer
}

func newDouDiZhuSeat(playerID qpTable.PlayerID, seatNumber qpTable.SeatNumber) (qpTable.QPSeat, error) {
	baseSeat := qpTable.NewQPSeat(playerID, seatNumber)

	pdkSeat := &DouDiZhuSeat{seatData: baseSeat}
	pdkSeat.CleanRoundData()

	return pdkSeat, nil
}

func NewDouDiZhuTable(tableNum int32, gameRuleCfg, tableCfg string) (*PokerDouDiZhuTable, int32, string) {
	baseTable, rspCode, err := qpTable.NewQPTable(tableNum, tableCfg, newDouDiZhuSeat)
	if err != nil {
		return nil, rspCode, err.Error()
	}

	t := PokerDouDiZhuTable{}
	rspCode, err = t.ParseTableOptConfig(gameRuleCfg)
	if rspCode != 0 {
		return nil, rspCode, err.Error()
	}

	baseTable.GameOverFunc = func() {
		t.handleXiaoJieSuan()
		t.handleDaJieSuan()
	}
	baseTable.Consumables = t.gameRule.Consumables * t.gameRule.MaxPlayer
	baseTable.TableRule.Table = &t
	t.table = baseTable
	t.table.SetMaxPlayers(t.gameRule.MaxPlayer)

	t.PaiMgr = pokerTable.NewPokerPaiBaseMgr(localConfig.GetConfig().IsTestPai)
	t.PaiMgr.XiPai(baseTable.MaxPlayers, 17)

	t.logic.SetRule(&t.gameRule)

	t.CleanRoundData()
	return &t, rspCode, ""
}
