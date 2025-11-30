package progress_bar

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ProgressBar struct {
	label   string
	total   int64
	current int64
	stop    chan struct{}
	wg      sync.WaitGroup
}

func NewProgressBar(label string, total int) *ProgressBar {
	p := &ProgressBar{label: label, total: int64(total), stop: make(chan struct{})}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		// initial draw
		p.draw(atomic.LoadInt64(&p.current))
		for {
			select {
			case <-p.stop:
				return
			case <-ticker.C:
				cur := atomic.LoadInt64(&p.current)
				if cur > p.total {
					cur = p.total
				}
				p.draw(cur)
			}
		}
	}()
	return p
}

func (p *ProgressBar) Inc() {
	for {
		cur := atomic.LoadInt64(&p.current)
		if cur >= p.total {
			return
		}
		if atomic.CompareAndSwapInt64(&p.current, cur, cur+1) {
			return
		}
		// retry on race
	}
}

func (p *ProgressBar) Done() {
	atomic.StoreInt64(&p.current, p.total)
	close(p.stop)
	p.wg.Wait()
	p.draw(p.total)
	_, _ = fmt.Fprintln(os.Stderr)
}

func (p *ProgressBar) draw(current int64) {
	if p.total <= 0 {
		return
	}
	percent := 0
	if p.total > 0 {
		percent = int(current * 100 / p.total)
	}
	barLen := 24
	filled := 0
	if p.total > 0 {
		filled = int(current * int64(barLen) / p.total)
	}
	if filled > barLen {
		filled = barLen
	}
	bar := strings.Repeat("#", filled) + strings.Repeat(" ", barLen-filled)
	_, _ = fmt.Fprintf(os.Stderr, "\r[%s] %4d/%-4d (%3d%%) %s", bar, current, p.total, percent, p.label)
}
