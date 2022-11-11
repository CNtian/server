package protoGameBasic

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type PlayerGameScore struct {
	UID     int64  `json:"uid" bson:"uid"`
	Nick    string `json:"nick" bson:"nick"`
	ClubID  int32  `json:"clubID" bson:"clubID"` // 玩家 所属俱乐部ID
	SScore  int64  `json:"sscore" bson:"score"`  // 得分(服务端使用)
	IsLeave int    `json:"leave" bson:"leave"`   // 是否提前撤离
}

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
	RoundID     primitive.ObjectID `json:"roundID"`
	TableID     int32              `json:"tableID"`
	Begin       time.Time          `json:"begin"`
	End         time.Time          `json:"end"`
	PlayerScore []*PlayerGameScore `json:"players"`
	GameID      int32              `json:"gameID"`   // 玩法ID
	GameName    string             `json:"gameName"` // 玩法名称

	ClubID       int32 `json:"clubID"`       // 大盟主俱乐部ID
	ClubPlayID   int64 `json:"clubPlayID"`   // 俱乐部玩法ID
	PayPlayerID  int64 `json:"payPlayerID"`  // 支付者ID
	ConsumeCount int32 `json:"consumeCount"` // 消耗数量

	RuleRound   int32 `json:"ruleRound"`   // 规则设置的局数
	ActualRound int32 `json:"actualRound"` // 实际玩的局数
}
