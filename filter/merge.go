package filter

import (
	"encoding/json"
)

// MergeJSON is
func MergeJSON(source, dst []byte) ([]byte, error) {
	var result map[string]interface{}
	err := json.Unmarshal(dst, &result)
	if err != nil {
		return nil, err
	}
	if len(source) == 0 {
		return dst, nil
	}

	var smap map[string]interface{}
	err = json.Unmarshal(source, &smap)
	if err != nil {
		return nil, err
	}

	err = MergeWithOverwrite(&result, smap)
	if err != nil {
		return nil, err
	}

	buf, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return nil, err
	}

	return buf, nil
}
