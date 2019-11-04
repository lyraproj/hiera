package hiera

import (
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
		var v px.Value
		if !value.Equals(px.Undef, nil) {
			// Convert value to rich data format
			ser := serialization.NewSerializer(c, px.EmptyMap)
			dc := px.NewCollector()
			ser.Convert(value, dc)
			v = dc.Value()
		}

		if renderAs == `yaml` {
			rf := c.Reflector().Reflect(v)
			var iv interface{}
			if rf.IsValid() && rf.CanInterface() {
				iv = rf.Interface()
			} else {
				iv = value.String()
			}
			bs, err := yaml.Marshal(iv)
			utils.WriteString(out, string(bs))
			if err != nil {
				panic(err)
			}
		} else {
			serialization.DataToJson(v, out)
		}
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
