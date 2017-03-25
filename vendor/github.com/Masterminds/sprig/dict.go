package sprig

func set(d map[string]interface{}, key string, value interface{}) map[string]interface{} {
	d[key] = value
	return d
}

func unset(d map[string]interface{}, key string) map[string]interface{} {
	delete(d, key)
	return d
}

func hasKey(d map[string]interface{}, key string) bool {
	_, ok := d[key]
	return ok
}

func pluck(key string, d ...map[string]interface{}) []interface{} {
	res := []interface{}{}
	for _, dict := range d {
		if val, ok := dict[key]; ok {
			res = append(res, val)
		}
	}
	return res
}

func keys(dict map[string]interface{}) []string {
	k := []string{}
	for key := range dict {
		k = append(k, key)
	}
	return k
}

func pick(dict map[string]interface{}, keys ...string) map[string]interface{} {
	res := map[string]interface{}{}
	for _, k := range keys {
		if v, ok := dict[k]; ok {
			res[k] = v
		}
	}
	return res
}

func omit(dict map[string]interface{}, keys ...string) map[string]interface{} {
	res := map[string]interface{}{}

	omit := make(map[string]bool, len(keys))
	for _, k := range keys {
		omit[k] = true
	}

	for k, v := range dict {
		if _, ok := omit[k]; !ok {
			res[k] = v
		}
	}
	return res
}

func dict(v ...interface{}) map[string]interface{} {
	dict := map[string]interface{}{}
	lenv := len(v)
	for i := 0; i < lenv; i += 2 {
		key := strval(v[i])
		if i+1 >= lenv {
			dict[key] = ""
			continue
		}
		dict[key] = v[i+1]
	}
	return dict
}
