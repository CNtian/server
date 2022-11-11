package qpTable

import (
	"math/rand"
	"qpGame/commonDefine/mateProto"
	"strconv"
	"time"
)

type QPSeat interface {
	CleanRoundData()
	GetSeatData() *SeatData
	GetXSeatData(int) interface{}
}

type GameScoreRec struct {
	Category int     `json:'category'`
	Score    float64 `json:"score"`
	BeiShu   int     `json:"bs"`
	//Target   string      `json:"name"`
	PaiXing interface{} `json:"px"`

	TargetUID []SeatNumber `json:"tUID"`
}

type SeatNumber int32
type SeatStatus uint32

const SS_Sitdown SeatStatus = 1           //1<<0	坐下来了
const SS_Ready SeatStatus = 2             //1<<1	已经准备
const SS_Playing SeatStatus = 4           //1<<2	在玩
const SS_Offline SeatStatus = 8           //1<<3	离线
const SS_Trusteeship SeatStatus = 16      //1<<4	托管
const SS_Looker SeatStatus = 32           //1<<4	观看
const SS_CustomDefineBase SeatStatus = 64 //自定义状态起始值

const INVALID_SEAT_NUMBER SeatNumber = -1 // 无效的座位号

// 新建一个座位
func NewQPSeat(id PlayerID, seatNum SeatNumber) *SeatData {
	playerData := NewPlayer(id)
	return &SeatData{
		Player:    playerData,
		Number:    seatNum,
		SeatScore: 0,
		Status:    SS_Sitdown,
		IsPlayed:  false,
	}
}

type SeatData struct {
	Player         *QPPlayer
	Number         SeatNumber
	SeatScore      float64
	Status         SeatStatus
	OperationStart int64
	ClubID         int32   // 所属俱乐部ID
	ClubScore      float64 // 俱乐部分
	Lat            float64 // 纬度
	Lng            float64 // 经度
	//MutexMap       map[int64]bool // 互斥人员
	IsPlayed bool // 是否玩过
	IsLeave  int  // 是否已经离开

	CurTuoGuanRound int32 // 当前已托管局数
	DissolveVote    int32 // 0:未操作  1:同意  2:不同意

	RoundOverMsg *mateProto.MessageMaTe // 小结算消息
	GameOverMsg  *mateProto.MessageMaTe // 大结算消息

	// 小局待清理
	StepCount        int32
	OperationID      string
	OperationIDBak   string
	RoundScore       float64
	GameScoreRecStep []GameScoreRec // 游戏分记录
}

// 清空每一小局数据
func (this *SeatData) CleanRoundData() {
	this.StepCount = 0
	this.OperationID, this.OperationIDBak = "", ""
	this.RoundScore = 0
	this.DelState(SS_Playing)
	this.GameScoreRecStep = make([]GameScoreRec, 0, 6)
}

func (this *SeatData) GetSeatData() *SeatData {
	return this
}

func (this *SeatData) GetXSeatData(int) interface{} {
	return nil
}

func (this *SeatData) PutGameScoreItem(v *GameScoreRec, sign float64) {
	t := *v
	t.Score *= sign
	this.GameScoreRecStep = append(this.GameScoreRecStep, t)
}

// 是否存在某种状态
func (this *SeatData) IsAssignSeatState(value SeatStatus) bool {
	if (this.Status & value) == value {
		return true
	}
	return false
}

func (this *SeatData) IsContainSeatState(value SeatStatus) bool {
	if (this.Status & value) != 0 {
		return true
	}
	return false
}

// 追加某项状态
func (this *SeatData) AppendState(value SeatStatus) {
	this.Status |= value
}

// 设置状态
func (this *SeatData) SetState(value SeatStatus) {
	this.Status = value
}

// 删除指定的状态
func (this *SeatData) DelState(value SeatStatus) {
	this.Status &= ^value
}

// 设置游戏分
func (this *SeatData) SetGameScore(value float64) {
	this.SeatScore = value
}

// 改变游戏分
func (this *SeatData) ChangeGameScore(value float64) {
	this.SeatScore += value
}

// 改变当局游戏中的分
func (this *SeatData) ChangeRoundScore(value float64) {
	this.RoundScore += value
}

// 创建操作ID
func (this *SeatData) MakeOperationID() {
	this.OperationIDBak = this.OperationID
	this.OperationStart = time.Now().Unix()
	this.OperationID = strconv.Itoa(int(this.Number)) + "_" +
		strconv.Itoa(int(this.StepCount)) + "_" +
		strconv.Itoa(rand.Intn(10))
	this.StepCount += 1
}

// 清理 操作ID
func (this *SeatData) CleanOperationID() {
	this.OperationIDBak = this.OperationID
	this.OperationID = ""
}

func (this *SeatData) GetOperationID() string {
	return this.OperationID
}
