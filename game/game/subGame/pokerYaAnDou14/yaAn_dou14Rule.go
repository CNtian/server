package pokerYaAnD14

import (
	"fmt"
	"qpGame/localConfig"
	"qpGame/qpTable"
	"strconv"
)

type Dou14Rule struct {
	RuleJson        string  `json:"-"`         //
	MaxRoundCount   int32   `json:"maxCount"`  // 总局数 6\12
	MaxPlayer       int32   `json:"maxPlayer"` // 最大人数 2\3人
	Multiple        string  `json:"multiple"`  // 倍数 0.5\1\2\3\4\5\10\20\30\50
	MultipleFloat64 float64 `json:"-"`

	//FengDingFanShu int32 `json:"FDFanShu"` // 封顶番数 8\16

	WangHua    int32 `json:"wangHua"` // 王花
	IsAnyHu    bool  `json:"anyHu"`   // 任意胡
	Dian7      bool  `json:"dian7"`   // 7当点
	LiuJuScore bool  `json:"liuJu"`   // 流局后检查，是否玩家报听

	// 服务器自定义(与客户端一致)
	Consumables int32 `json:"consumables"` // 消耗(房卡|钻石)
}

func (this *Dou14Rule) CheckField() error {
	switch this.MaxRoundCount {
	case 6, 8:
		this.Consumables = 3 // 1
	case 16:
		this.Consumables = 6 // 2
	default:
		return fmt.Errorf("maxRound := %d", this.MaxRoundCount)
	}

	if this.MaxPlayer < 2 || this.MaxPlayer > 3 {
		return fmt.Errorf("maxPlayer := %d", this.MaxPlayer)
	}

	//switch this.FengDingFanShu {
	//case 2, 4, 8:
	//default:
	//	return fmt.Errorf("FDFanShu := %d", this.FengDingFanShu)
	//}

	switch this.WangHua {
	case 3, 6, 9, 12, 15, 18:
	default:
		return fmt.Errorf("wangHua := %d", this.WangHua)
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

func (this *Dou14Rule) GetMaxPlayerCount() int32 {
	return this.MaxPlayer
}

func newPaoDeKuaiSeat(playerID qpTable.PlayerID, seatNumber qpTable.SeatNumber) (qpTable.QPSeat, error) {
	baseSeat := qpTable.NewQPSeat(playerID, seatNumber)

	pdkSeat := &PokerDou14Seat{seatData: baseSeat}
	pdkSeat.CleanRoundData()

	return pdkSeat, nil
}

func NewDou14Table(tableNum int32, gameRuleCfg, tableCfg string) (*PokerDou14Table, int32, string) {
	baseTable, rspCode, err := qpTable.NewQPTable(tableNum, tableCfg, newPaoDeKuaiSeat)
	if err != nil {
		return nil, rspCode, err.Error()
	}

	t := PokerDou14Table{}
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

	t.PaiMgr = NewDou14PaiBaseMgr(localConfig.GetConfig().IsTestPai)
	t.PaiMgr.XiPai(t.gameRule.WangHua)

	t.logic.SetRule(&t.gameRule)

	t.bankerSeatNo = qpTable.INVALID_SEAT_NUMBER
	t.huSeatNo = qpTable.INVALID_SEAT_NUMBER
	t.CleanRoundData()
	return &t, rspCode, ""
}
