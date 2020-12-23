package cache

type Cache interface {
	Get(x interface{}) (interface{}, bool)
	Add(key, value interface{})
	Keys() []interface{}
	Delete(key interface{})
}
