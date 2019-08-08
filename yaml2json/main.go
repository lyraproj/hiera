// Utility program to convert yaml on stdin to formatted json on stdout
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func stringKeys(x interface{}) interface{} {
	switch x := x.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range x {
			m[k.(string)] = stringKeys(v)
		}
		return m
	case []interface{}:
		for i, v := range x {
			x[i] = stringKeys(v)
		}
	}
	return x
}

func main() {
	data, err := ioutil.ReadAll(os.Stdin)
	if err == nil {
		var body interface{}
		if err = yaml.Unmarshal(data, &body); err == nil {
			var bytes []byte
			if bytes, err = json.MarshalIndent(stringKeys(body), ``, ` `); err == nil {
				if _, err = os.Stdout.Write(bytes); err == nil {
					return
				}
			}
		}
	}
	log.Fatal(err.Error())
}
