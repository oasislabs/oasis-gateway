package redis

type op string

type command interface {
	Op() op
	Keys() []string
	Args() []interface{}
}

const (
	mqnext     op = "return mqnext(KEYS[1])"
	mqinsert   op = "return mqinsert(KEYS[1], ARGV[1], ARGV[2], ARGV[3])"
	mqretrieve op = "return mqretrieve(KEYS[1], ARGV[1], ARGV[2])"
	mqdiscard  op = "return mqdiscard(KEYS[1], ARGV[1], ARGV[2], ARGV[3])"
	mqremove   op = "return mqremove(KEYS[1])"
)

type nextRequest struct {
	Key string
}

func (r nextRequest) Op() op {
	return mqnext
}

func (r nextRequest) Keys() []string {
	return []string{r.Key}
}

func (r nextRequest) Args() []interface{} {
	return nil
}

type insertRequest struct {
	Offset  uint64
	Key     string
	Content string
	Type    string
}

func (r insertRequest) Op() op {
	return mqinsert
}

func (r insertRequest) Keys() []string {
	return []string{r.Key}
}

func (r insertRequest) Args() []interface{} {
	return []interface{}{r.Offset, r.Type, r.Content}
}

type retrieveRequest struct {
	Count  uint
	Offset uint64
	Key    string
}

func (r retrieveRequest) Op() op {
	return mqretrieve
}

func (r retrieveRequest) Keys() []string {
	return []string{r.Key}
}

func (r retrieveRequest) Args() []interface{} {
	return []interface{}{r.Offset, r.Count}
}

type discardRequest struct {
	KeepPrevious bool
	Count        uint
	Offset       uint64
	Key          string
}

func (r discardRequest) Op() op {
	return mqdiscard
}

func (r discardRequest) Keys() []string {
	return []string{r.Key}
}

func (r discardRequest) Args() []interface{} {
	return []interface{}{r.Offset, r.Count, r.KeepPrevious}
}

type removeRequest struct {
	Key string
}

func (r removeRequest) Op() op {
	return mqremove
}

func (r removeRequest) Keys() []string {
	return []string{r.Key}
}

func (r removeRequest) Args() []interface{} {
	return nil
}
