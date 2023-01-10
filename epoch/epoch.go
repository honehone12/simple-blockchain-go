package epoch

type Epoch struct {
	f func()
	c chan bool
}

func NewEpoch(f func()) *Epoch {
	return &Epoch{
		f: f,
		c: make(chan bool),
	}
}

func (e *Epoch) C() chan<- bool {
	return e.c
}

func (e *Epoch) StartEpochRoutine() {
	for flg := range e.c {
		if flg {
			e.f()
		} else {
			break
		}
	}
	close(e.c)
}
