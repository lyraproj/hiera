package hiera

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/lyraproj/pcore/serialization"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"gopkg.in/yaml.v3"
)

type RenderName string

const (
	YAML   = RenderName(`yaml`)
	JSON   = RenderName(`json`)
	Binary = RenderName(`binary`)
	Text   = RenderName(`s`)
)

func Render(c px.Context, renderAs RenderName, value px.Value, out io.Writer) {
	switch renderAs {
	case YAML, JSON:
		var v interface{}
		if !value.Equals(px.Undef, nil) {
			// Convert value to rich data format
			ser := serialization.NewSerializer(c, px.EmptyMap)
			dc := px.NewCollector()
			ser.Convert(value, dc)
			rf := c.Reflector().Reflect(dc.Value())
			if rf.IsValid() && rf.CanInterface() {
				v = rf.Interface()
			} else {
				v = value.String()
			}
		}
		var bs []byte
		var err error
		if renderAs == `yaml` {
			bs, err = yaml.Marshal(v)
		} else {
			// JSON is not able to handle the result of reflecting an empty untyped map.
			if em, ok := v.(map[interface{}]interface{}); ok && len(em) == 0 {
				v = map[string]interface{}{}
			}
			bs, err = json.Marshal(v)
		}
		if err != nil {
			panic(err)
		}
		utils.WriteString(out, string(bs))
	case Binary:
		bi := types.CoerceTo(c, `lookup value`, types.DefaultBinaryType(), value).(*types.Binary)
		_, err := out.Write(bi.Bytes())
		if err != nil {
			panic(err)
		}
	case Text:
		utils.Fprintln(out, value)
	default:
		panic(fmt.Errorf(`unknown rendering '%s'`, renderAs))
	}
}
