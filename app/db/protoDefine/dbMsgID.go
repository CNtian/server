package protoDefine

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type JiangLiDetail struct {
	ClubID      int32  `json:"-" bson:"got_clubID"`
	ClubCreator int64  `json:"-" bson:"got_clubCID"`
	Percentage  string `json:"-" bson:"percentage"` // 百分比
	JiangLi     int64  `json:"-" bson:"jiang_li"`

	CurClubScore int64 `json:"-" bson:"-"`
}

type PlayerGameScore struct {
	UID         int64  `json:"uid" bson:"uid"`
	Nick        string `json:"nick" bson:"nick"`
	ClubID      int32  `json:"clubID" bson:"clubID"`            // 玩家 所属俱乐部ID
	ClubCreator int64  `json:"clubCreator" bson:"club_creator"` // 俱乐部创建者
	ClubName    string `json:"clubName" bson:"clubName"`        // 玩家 所属俱乐部名称
	SScore      int64  `json:"sscore" bson:"score"`             // 得分(服务端使用)
	IsLeave     int    `json:"leave" bson:"leave"`              // 是否提前撤离

	HaoKa      int64            `json:"-" bson:"haoKa"`
	XiaoHao    int64            `json:"-" bson:"xiaoHao"`
	GongXian   int64            `json:"-" bson:"gongXian"`
	JiangLiArr []*JiangLiDetail `json:"-" bson:"JLDetail"`
	BaoDi      int64            `json:"-" bson:"bao_di"` // 保底

	ScoreText   string                   `json:"score" bson:"-"` // 得分(客户端使用)
	IsMaxWinner bool                     `json:"-" bson:"-"`
	JiangLiMap  map[int32]*JiangLiDetail `json:"-" bson:"-"`

	//CurClubScore int64 `json:"-" bson:"-"` // 俱乐部分
}
type SortPlayerGameScore []*PlayerGameScore

func (a SortPlayerGameScore) Len() int           { return len(a) }
func (a SortPlayerGameScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortPlayerGameScore) Less(i, j int) bool { return a[i].SScore > a[j].SScore }

// 小局结束
const ID_RoundOver = 601

type SS_RoundRecord struct {
	RoundID primitive.ObjectID `json:"roundID"`
	ClubID  int32              `json:"clubID"` // 盟主俱乐部ID
	TableID int32              `json:"tableID"`
	Begin   time.Time          `json:"begin"`
	End     time.Time          `json:"end"`

	CurRound int32              `json:"curRound" `
	Players  []*PlayerGameScore `json:"players"`
	GameStep string             `json:"gameStep"`
}

// 大局结束
const ID_GameOver = 602

type SS_GameOverRecord struct {
	RoundID     primitive.ObjectID  `json:"roundID"`
	TableID     int32               `json:"tableID"`
	Begin       time.Time           `json:"begin"`
	End         time.Time           `json:"end"`
	PlayerScore SortPlayerGameScore `json:"players"`
	GameID      int32               `json:"gameID"`   // 玩法ID
	GameName    string              `json:"gameName"` // 玩法名称

	MZClubID     int32 `json:"clubID"`       // 大盟主俱乐部ID
	ClubPlayID   int64 `json:"clubPlayID"`   // 俱乐部玩法ID
	PayPlayerID  int64 `json:"payPlayerID"`  // 支付者ID
	ConsumeCount int32 `json:"consumeCount"` // 消耗数量
	RuleRound    int32 `json:"ruleRound"`    // 规则设置的局数
	ActualRound  int32 `json:"actualRound"`  // 实际玩的局数
}

// 605

// 统计管理费
const ID_TotalMangeFee = 606

type SS_TotalMangeFee struct {
	Date int `json:"date"`
}

// 删除过期数据
const ID_DeleteExpiredData = 607

// 日活写入
const ID_WriteDaily = 608
