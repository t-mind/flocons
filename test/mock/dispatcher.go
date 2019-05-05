package mock

type NullDispatcher struct{}

func (*NullDispatcher) Get(name string) (string, error) { return "", nil }
func (*NullDispatcher) Set(backends []string) error     { return nil }
func (*NullDispatcher) Clear()                          {}
