package vm

import (
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
)

func parseYAMLDocuments(reader io.Reader) ([]interface{}, error) {
	ret := []interface{}{}
	d := yaml.NewYAMLToJSONDecoder(reader)
	for {
		var doc interface{}
		if err := d.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if doc != nil {
			ret = append(ret, doc)
		}
	}
	return ret, nil
}
