package c

import (
	"context"
	"sync"
	"time"
)

// SSE 断线续传缓冲层。
//
// 生成过程不再绑定在 HTTP 连接上：一轮回答写进 streamRun 的事件缓冲区，
// 浏览器 SSE 断开后服务端继续生成；重新连接时带上最后收到的序号，
// 把漏掉的事件补发出来再接着实时推送，用户看到的是「继续输出」而不是「回复失败」。
const (
	// streamBufferLimit 是一轮回答最多缓存的事件数（逐字增量，几千条足够一次回答）
	streamBufferLimit = 6000
	// streamRunTTL 是回答结束后缓冲区保留时间，留给晚到的重连
	streamRunTTL = 2 * time.Minute
	// streamSubscriberBuffer 是单个订阅者通道容量，满了只丢该订阅者，不阻塞生成
	streamSubscriberBuffer = 512
)

type streamEvent struct {
	seq  int
	name string
	data []byte
}

type streamRun struct {
	mu      sync.Mutex
	events  []streamEvent
	subs    map[int]chan streamEvent
	nextSub int
	seq     int
	done    bool
	cancel  context.CancelFunc
}

func newStreamRun() *streamRun {
	return &streamRun{subs: make(map[int]chan streamEvent)}
}

// publish 追加事件并广播给所有订阅者；订阅者通道满时跳过该订阅者。
func (r *streamRun) publish(name string, data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	event := streamEvent{seq: r.seq, name: name, data: data}
	// 环形上限：超过后丢弃最早的事件，避免长回答无限增长
	if len(r.events) >= streamBufferLimit {
		r.events = append(r.events[:0], r.events[len(r.events)/2:]...)
	}
	r.events = append(r.events, event)
	for _, ch := range r.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

// finish 标记本轮结束并关闭所有订阅者通道。
func (r *streamRun) finish() {
	r.mu.Lock()
	if r.done {
		r.mu.Unlock()
		return
	}
	r.done = true
	for id, ch := range r.subs {
		close(ch)
		delete(r.subs, id)
	}
	r.mu.Unlock()
}

// subscribe 从 afterSeq 之后开始订阅，返回积压事件与实时通道。
func (r *streamRun) subscribe(afterSeq int) ([]streamEvent, chan streamEvent, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var backlog []streamEvent
	for _, event := range r.events {
		if event.seq > afterSeq {
			backlog = append(backlog, event)
		}
	}
	if r.done {
		return backlog, nil, true
	}
	id := r.nextSub
	r.nextSub++
	ch := make(chan streamEvent, streamSubscriberBuffer)
	r.subs[id] = ch
	return backlog, ch, false
}

// unsubscribe 移除订阅者。
func (r *streamRun) unsubscribe(ch chan streamEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, candidate := range r.subs {
		if candidate == ch {
			delete(r.subs, id)
			close(ch)
			return
		}
	}
}

// streamHub 按「用户 + 会话」保存正在进行（或刚结束）的一轮回答。
type streamHub struct {
	mu   sync.Mutex
	runs map[string]*streamRun
}

func newStreamHub() *streamHub {
	return &streamHub{runs: make(map[string]*streamRun)}
}

func (h *streamHub) key(userID, conversationID uint) string {
	return itoa(userID) + ":" + itoa(conversationID)
}

// get 取出当前会话的缓冲（可能为空）。
func (h *streamHub) get(key string) *streamRun {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.runs[key]
}

// start 注册一轮新回答，结束后延迟清理，留给断线重连补发。
func (h *streamHub) start(key string) *streamRun {
	run := newStreamRun()
	h.mu.Lock()
	if old, ok := h.runs[key]; ok {
		old.finish()
	}
	h.runs[key] = run
	h.mu.Unlock()
	return run
}

// removeAfter 结束一段时间后删除缓冲，若期间已经有新的一轮则保留新的。
func (h *streamHub) removeAfter(key string, run *streamRun, delay time.Duration) {
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		<-timer.C
		h.mu.Lock()
		if current, ok := h.runs[key]; ok && current == run {
			delete(h.runs, key)
		}
		h.mu.Unlock()
	}()
}

// itoa 轻量整数转字符串，避免为了拼接 key 引入 fmt。
func itoa(value uint) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for value > 0 {
		pos--
		buf[pos] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[pos:])
}

// setCancel 记录本轮生成的取消函数。
func (r *streamRun) setCancel(cancel context.CancelFunc) {
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()
}

// cancelRun 取消本轮生成（用户已长时间断开时调用）。
func (r *streamRun) cancelRun() {
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// hasSubscribers 表示当前还有 SSE 连接在监听。
func (r *streamRun) hasSubscribers() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.subs) > 0
}

// state 返回本轮是否仍在生成，以及已产生的最大事件序号。
func (r *streamRun) state() (bool, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.done, r.seq
}

// state 按 key 查询；没有缓冲时表示当前没有进行中的回答。
func (h *streamHub) state(key string) (bool, int) {
	h.mu.Lock()
	run := h.runs[key]
	h.mu.Unlock()
	if run == nil {
		return false, 0
	}
	return run.state()
}
