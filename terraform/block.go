package terraform

type Block interface {
	Ref() Value
	Template() string
}
