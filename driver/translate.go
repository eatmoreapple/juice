package driver

type Translate interface {
	Translate(matched string) string
}

type TranslateFunc func(matched string) string

func (f TranslateFunc) Translate(matched string) string {
	return f(matched)
}
