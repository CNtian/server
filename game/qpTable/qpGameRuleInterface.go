package qpTable

type QPGameRule interface {
	GetMaxPlayerCount() int32
	CheckField() error
}
