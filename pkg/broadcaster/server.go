package broadcaster

// Server defines the functions required to create a new
// channel broadcaster
type Server[T any] struct {
	source         chan T
	listeners      []chan T
	addListener    chan chan T
	removeListener chan (<-chan T)
	shutdown       chan bool
}

// New creates a new fully configured Server
func New[T any]() *Server[T] {
	server := &Server[T]{
		shutdown:       make(chan bool),
		source:         make(chan T),
		listeners:      make([]chan T, 0),
		addListener:    make(chan chan T),
		removeListener: make(chan (<-chan T)),
	}

	go server.serve()

	return server
}

// Publish consumer a payload
func (s *Server[T]) Publish(payload T) {
	s.source <- payload
}

// Subscribe adds a new subscriber channel
func (s *Server[T]) Subscribe() <-chan T {
	c := make(chan T)
	s.addListener <- c
	return c
}

// Unsubscribe removes a subscriber channel
func (s *Server[T]) Unsubscribe(c <-chan T) {
	s.removeListener <- c
}

// ActiveSubscribers returns the number of current subscribers
func (s *Server[T]) ActiveSubscribers() int {
	return len(s.listeners)
}

// Shutdown will stop the broadcaster
func (s *Server[T]) Shutdown() {
	s.shutdown <- true
}

// serve listens for incoming payloads, new subscribers and handles
// unsubscribes
func (s *Server[T]) serve() {
	defer func() {
		for _, listener := range s.listeners {
			if listener != nil {
				close(listener)
			}
		}
	}()

	for {
		select {
		case <-s.shutdown:
			return
		case newListener := <-s.addListener:
			s.listeners = append(s.listeners, newListener)
		case listenerToRemove := <-s.removeListener:
			for i, ch := range s.listeners {
				if ch == listenerToRemove {
					s.listeners[i] = s.listeners[len(s.listeners)-1]
					s.listeners = s.listeners[:len(s.listeners)-1]
					close(ch)
					break
				}
			}
		case val, ok := <-s.source:
			if !ok {
				return
			}
			for _, listener := range s.listeners {
				if listener != nil {
					listener <- val
				}
			}
		}
	}
}
