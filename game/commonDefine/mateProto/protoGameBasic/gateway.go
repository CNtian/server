package protoGameBasic

// 玩家登陆网关
const ID_PlayerNetStatus = 101

type PlayerNetStatus struct {
	IsOnline bool `json:"isOnline"`
}

// 玩家未在游戏中
const ID_PlayerNotInGame = 800
