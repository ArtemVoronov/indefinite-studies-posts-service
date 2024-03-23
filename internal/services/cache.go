package services

func GetFromCache(key string) (string, error) {
	return Instance().Cache().Get(key)
}

func PutToCache(key, value string) error {
	cache := Instance().Cache()
	return cache.Set(key, value, cache.PostsTTL)
}
