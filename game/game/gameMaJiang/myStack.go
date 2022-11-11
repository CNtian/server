package gameMaJiang

type Stack struct {
	data     []int8
	curIndex int
}

func (this *Stack) Init(size int32) {
	this.data = make([]int8, 0, size)
	this.curIndex = 0
}

func (this *Stack) Push(value int8) {
	if this.curIndex < len(this.data) {
		this.data[this.curIndex] = value
	} else {
		this.data = append(this.data, value)
	}
	this.curIndex += 1
}

func (this *Stack) PushMul(value int8, times int8) {
	for i := times; i > 0; i-- {
		this.Push(value)
	}
}

func (this *Stack) Pop(times int) {
	if this.curIndex-times >= 0 {
		this.curIndex -= times
	} else {
		this.curIndex = 0
	}
}

func (this *Stack) GetPaiArr() []int8 {
	return this.data
}
