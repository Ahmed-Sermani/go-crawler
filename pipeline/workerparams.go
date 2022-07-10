package pipeline

// compile time check that WorkerParams implements StageParams
var _ StageParams = (*WorkerParams)(nil)

type WorkerParams struct {
    Stage int

    InCh <-chan Payload
    OutCh chan<- Payload
    ErrCh chan<- error
}

func (wp *WorkerParams) StageIndex() int { return wp.Stage }
func (wp *WorkerParams) Input() <-chan Payload { return wp.InCh }
func (wp *WorkerParams) Output() chan<- Payload { return wp.OutCh }
func (wp *WorkerParams) Error() chan<- error { return wp.ErrCh }
