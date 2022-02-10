package uicfg

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var example = `{"name":"pvna01","version":"0.0","date":1644326235,"aud":"https://static.practable.io/ui/pvna-1.0","images":[{"for":"dut","src":"https://assets.practable.io/images/experiments/pvna01-0.0/dut.png","alt":"PocketVNA with 4-way RF switch and 80mm antenna","figcaption":"Photo of the DUT (80mm Antenna)","width":300,"height":480},{"for":"sparam","src":"https://assets.practable.io/images/experiments/pvna01-0.0/sparam.svg","alt":"Line drawing of one port measurement showing stimulus (in), and response (out)","figcaption":"Diagram of one-port (S11) measurement","width":300,"height":480}],"parameters":[{"for":"ui","are":[{"k":"title","v":"80mm Antenna"}]},{"for":"dut","are":[{"k":"name","v":"80mm Antenna"},{"k":"d1","v":"80mm"},{"k":"d2","v":"10mm"},{"k":"d3","v":"10mm"},{"k":"d4","v":"15mm"},{"k":"d5","v":"10mm"},{"k":"c1","v":"1pF"},{"k":"c2","v":"2pF"}]},{"for":"board","are":[{"k":"material","v":"FR4"},{"k":"thickness","v":"1.6mm"}]}]}`

func TestModel(t *testing.T) {

	var c Config
	err := json.Unmarshal([]byte(example), &c)
	assert.NoError(t, err)

	assert.Equal(t, c.Aud, "https://static.practable.io/ui/pvna-1.0")

	var doneImageDUT, doneImageSparam bool

	for _, img := range c.Images {
		if img.For == "dut" {
			assert.Equal(t, img.Src, "https://assets.practable.io/images/experiments/pvna01-0.0/dut.png")
			assert.Equal(t, img.Width, 300)
			assert.Equal(t, img.Height, 480)
			assert.Equal(t, img.Alt, "PocketVNA with 4-way RF switch and 80mm antenna")
			assert.Equal(t, img.FigCaption, "Photo of the DUT (80mm Antenna)")
			doneImageDUT = true
		} else if img.For == "sparam" {
			assert.Equal(t, img.Src, "https://assets.practable.io/images/experiments/pvna01-0.0/sparam.svg")
			assert.Equal(t, img.Width, 300)
			assert.Equal(t, img.Height, 480)
			assert.Equal(t, img.Alt, "Line drawing of one port measurement showing stimulus (in), and response (out)")
			assert.Equal(t, img.FigCaption, "Diagram of one-port (S11) measurement")
			doneImageSparam = true
		} else {
			t.Error("incorrect image present - has json been unmarshalled correctly? Has example been changed?")
		}
	} //for

	assert.True(t, doneImageDUT && doneImageSparam, "did not find both images in the config object")

	var donePUI, donePDUT, donePBoard bool
	var doneKVD4, doneKVThickness bool

	for _, p := range c.Parameters {

		switch {

		case p.For == "ui":
			donePUI = true
			for _, kv := range p.Are { //only one entry
				assert.Equal(t, kv.K, "title")
				assert.Equal(t, kv.V, "80mm Antenna")
			}

		case p.For == "dut":
			donePDUT = true
			for _, kv := range p.Are {
				if kv.K == "d4" {
					doneKVD4 = true
					assert.Equal(t, kv.V, "15mm")
				}
			}

		case p.For == "board":
			donePBoard = true
			for _, kv := range p.Are {
				if kv.K == "thickness" {
					doneKVThickness = true
					assert.Equal(t, kv.V, "1.6mm")
				}
			}
		}

	}

	assert.True(t, donePUI && donePDUT && donePBoard, "did not find all three parameter sets in the config object")
	assert.True(t, doneKVD4 && doneKVThickness, "some selected parameters were found to be incorrect")

}
