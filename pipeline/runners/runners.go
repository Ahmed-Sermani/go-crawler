/*
  Contains buildin StageRunner Implementations
*/

package runners

func emitError(err error, errCh chan<- error) {
	select {
	case errCh <- err:
	default: // error channel is full.
	}
}
